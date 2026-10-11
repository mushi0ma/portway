package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"portway/internal/execution"
	"portway/internal/identity"
	"portway/internal/problem"
	"portway/internal/secure"
)

const maxEventStreams = 4

func isEventStream(r *http.Request) bool {
	p := strings.Split(r.URL.Path, "/")
	return r.Method == "GET" && len(p) == 5 && p[1] == "v1" && p[2] == "runs" && secure.UUID(p[3]) && p[4] == "events"
}

func eventCursor(r *http.Request, run string) (int64, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	h := r.Header.Values("Last-Event-ID")
	if err != nil || len(q) > 1 || len(h) > 1 || len(q) == 1 && (len(q["cursor"]) != 1 || len(h) != 0) {
		return 0, problem.New(400, "INVALID_CURSOR", "Use one cursor query or Last-Event-ID header")
	}
	if len(q) == 1 {
		return execution.ParseCursor(run, q.Get("cursor"))
	}
	if len(h) == 1 {
		return execution.ParseCursor(run, h[0])
	}
	return 0, nil
}

func (s *Server) executionEvents(w http.ResponseWriter, r *http.Request, _ identity.Principal) (int, any, error) {
	if r.Method != http.MethodGet {
		return 0, nil, problem.New(405, "METHOD_NOT_ALLOWED", "Use GET for event streaming")
	}
	after, err := eventCursor(r, r.PathValue("id"))
	if err != nil {
		return 0, nil, err
	}
	if s.streams.Add(1) > maxEventStreams {
		s.streams.Add(-1)
		w.Header().Set("Retry-After", "2")
		return 0, nil, problem.New(429, "STREAM_LIMIT", "Too many open event streams")
	}
	defer s.streams.Add(-1)
	load := func(ctx context.Context, seq int64) (execution.HistoryPage, error) {
		return s.executionHistory(ctx, r, seq)
	}
	// Validate identity, ownership, high-water and retention before sending 200.
	page, err := load(r.Context(), after)
	if err != nil {
		return 0, nil, err
	}
	streamHistory(w, r, after, page, load)
	return 0, nil, nil
}

func (s *Server) executionHistory(parent context.Context, r *http.Request, after int64) (execution.HistoryPage, error) {
	var out struct {
		Data  *execution.HistoryPage `json:"data"`
		Error *problem.Error         `json:"error"`
	}
	if s.Config.ExecutionURL == "" || len(s.Config.ExecutionKey) < 32 {
		return execution.HistoryPage{}, problem.Unavailable()
	}
	// Each poll revalidates the JWT and device through execution -> control.
	// No DB connection is retained while waiting for the next page or client.
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", s.Config.ExecutionURL+"/internal"+r.URL.Path+"?after="+strconv.FormatInt(after, 10), nil)
	if err != nil {
		return execution.HistoryPage{}, problem.Unavailable()
	}
	req.Header.Set("Authorization", r.Header.Get("Authorization"))
	req.Header.Set("X-Device-Session", r.Header.Get("X-Device-Session"))
	req.Header.Set("X-Portway-Service-Key", s.Config.ExecutionKey)
	client := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return execution.HistoryPage{}, problem.Unavailable()
	}
	defer response.Body.Close()
	b, err := io.ReadAll(io.LimitReader(response.Body, (2<<20)+1))
	if err != nil || len(b) > 2<<20 || json.Unmarshal(b, &out) != nil {
		return execution.HistoryPage{}, problem.Unavailable()
	}
	if response.StatusCode >= 400 && response.StatusCode <= 599 && out.Error != nil {
		out.Error.Status = response.StatusCode
		return execution.HistoryPage{}, out.Error
	}
	if response.StatusCode != 200 || out.Data == nil || !validHistory(*out.Data, r.PathValue("id"), after) {
		return execution.HistoryPage{}, problem.Unavailable()
	}
	return *out.Data, nil
}

func validHistory(p execution.HistoryPage, run string, after int64) bool {
	if len(p.Items) > execution.HistoryPageSize || p.PrunedThrough < 0 || p.PrunedThrough > after || p.SnapshotSeq < after+int64(len(p.Items)) {
		return false
	}
	for i, e := range p.Items {
		if e.RunID != run || e.Seq != after+int64(i)+1 || e.SchemaVersion != 1 || !secure.UUID(e.ID) || !secure.UUID(e.CorrelationID) || e.Type == "" || !json.Valid(e.Payload) {
			return false
		}
		for _, ch := range e.Type {
			if !(ch >= 'a' && ch <= 'z' || ch == '.' || ch == '_') {
				return false
			}
		}
	}
	return len(p.Items) == execution.HistoryPageSize || p.SnapshotSeq == after+int64(len(p.Items))
}

// Errors after the initial response close the stream. Reconnect obtains an HTTP
// status (401/410/503); transport failures never invent durable business events.
func streamHistory(w http.ResponseWriter, r *http.Request, after int64, page execution.HistoryPage, load func(context.Context, int64) (execution.HistoryPage, error)) {
	controller := http.NewResponseController(w)
	if controller.SetWriteDeadline(time.Time{}) != nil {
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	write := func(b []byte) bool {
		if r.Context().Err() != nil || controller.SetWriteDeadline(time.Now().Add(2*time.Second)) != nil {
			return false
		}
		if _, err := w.Write(b); err != nil || controller.Flush() != nil {
			return false
		}
		return controller.SetWriteDeadline(time.Time{}) == nil
	}
	if !write([]byte(": connected\nretry: 1000\n\n")) {
		return
	}
	lastWrite := time.Now()
	for {
		if len(page.Items) > 0 {
			var b bytes.Buffer
			for _, e := range page.Items {
				encoded, err := json.Marshal(e)
				if err != nil {
					return
				}
				fmt.Fprintf(&b, "id: %s\nevent: %s\ndata: %s\n\n", execution.Cursor(e.RunID, e.Seq), e.Type, encoded)
				after = e.Seq
			}
			if !write(b.Bytes()) {
				return
			}
			lastWrite = time.Now()
		} else if time.Since(lastWrite) >= 10*time.Second {
			if !write([]byte(": heartbeat\n\n")) {
				return
			}
			lastWrite = time.Now()
		}
		if len(page.Items) < execution.HistoryPageSize {
			timer := time.NewTimer(time.Second)
			select {
			case <-r.Context().Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
		var err error
		page, err = load(r.Context(), after)
		if err != nil {
			return
		}
	}
}
