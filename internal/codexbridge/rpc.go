package codexbridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// rpc is a minimal stdio JSON-RPC client for `codex app-server` (one process per operation).
type rpc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	cancel context.CancelFunc
	wmu    sync.Mutex
	mu     sync.Mutex
	next   int
	wait   map[int]chan rpcReply
	notes  chan rpcNote
	done   chan struct{}
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type rpcReply struct {
	Result json.RawMessage
	Err    *rpcError
}
type rpcNote struct {
	Method string
	Params json.RawMessage
}

func (e *rpcError) Error() string { return "codex app-server error " + strconv.Itoa(e.Code) }

var errRPCClosed = errors.New("codex app-server closed")

func startRPC(parent context.Context, binary, dir string, env []string) (*rpc, error) {
	ctx, cancel := context.WithCancel(parent)
	cmd := native(ctx, binary, dir, env, "app-server")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	c := &rpc{cmd: cmd, stdin: stdin, cancel: cancel, wait: map[int]chan rpcReply{}, notes: make(chan rpcNote, 64), done: make(chan struct{})}
	go c.read(stdout)
	var init json.RawMessage
	if err = c.call(ctx, "initialize", map[string]any{"clientInfo": map[string]string{"name": "portway_bridge", "title": "Portway bridge", "version": "2.0.0"}}, &init); err != nil {
		c.Close()
		return nil, err
	}
	if err = c.send(map[string]any{"method": "initialized"}); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}
func (c *rpc) read(stdout io.Reader) {
	defer close(c.done)
	scanner := bufio.NewScanner(io.LimitReader(stdout, 64<<20))
	scanner.Buffer(make([]byte, 64<<10), 8<<20)
	for scanner.Scan() {
		var m struct {
			ID     *json.RawMessage `json:"id"`
			Method string           `json:"method"`
			Params json.RawMessage  `json:"params"`
			Result json.RawMessage  `json:"result"`
			Error  *rpcError        `json:"error"`
		}
		if json.Unmarshal(scanner.Bytes(), &m) != nil {
			continue
		}
		switch {
		case m.Method != "" && m.ID != nil:
			// The bridge never grants interactive approvals or tools to app-server.
			_ = c.send(map[string]any{"id": m.ID, "error": map[string]any{"code": -32601, "message": "unsupported by Portway bridge"}})
		case m.Method != "":
			select {
			case c.notes <- rpcNote{Method: m.Method, Params: m.Params}:
			default:
			}
		case m.ID != nil:
			id, err := strconv.Atoi(string(*m.ID))
			if err != nil {
				continue
			}
			c.mu.Lock()
			ch := c.wait[id]
			delete(c.wait, id)
			c.mu.Unlock()
			if ch != nil {
				ch <- rpcReply{Result: m.Result, Err: m.Error}
			}
		}
	}
}
func (c *rpc) send(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_, err = c.stdin.Write(append(raw, '\n'))
	return err
}

// call sends one request; params==nil omits the field (as some methods expect).
func (c *rpc) call(ctx context.Context, method string, params, out any) error {
	c.mu.Lock()
	c.next++
	id := c.next
	ch := make(chan rpcReply, 1)
	c.wait[id] = ch
	c.mu.Unlock()
	msg := map[string]any{"method": method, "id": id}
	if params != nil {
		msg["params"] = params
	}
	if err := c.send(msg); err != nil {
		return errRPCClosed
	}
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case r := <-ch:
		if r.Err != nil {
			return r.Err
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(r.Result, out)
	case <-c.done:
		return errRPCClosed
	case <-timer.C:
		return errors.New("codex app-server timeout")
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (c *rpc) Close() {
	_ = c.stdin.Close()
	select {
	case <-c.done:
	case <-time.After(2 * time.Second):
	}
	c.cancel()
	_ = c.cmd.Wait()
}
