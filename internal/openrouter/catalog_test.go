package openrouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func catalog(pricing string) string {
	return `{"data":[{"id":"vendor/model:free","name":"fixture","context_length":32768,"supported_parameters":["tools","tool_choice"],"architecture":{"input_modalities":["text"],"output_modalities":["text"]},"pricing":` + pricing + `}]}`
}
func TestFreeCatalog(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		want      int
	}{
		{"zero", catalog(`{"prompt":"0","completion":"0","request":"0"}`), 1},
		{"decimal", catalog(`{"prompt":"0.0000","completion":"0e-12"}`), 1},
		{"paid-input", catalog(`{"prompt":"0.00000000001","completion":"0"}`), 0},
		{"paid-hidden", catalog(`{"prompt":"0","completion":"0","internal_reasoning":"0.1"}`), 0},
		{"paid-cache", catalog(`{"prompt":"0","completion":"0","input_cache_read":"0.01"}`), 0},
		{"unknown-price", catalog(`{"prompt":"0"}`), 0},
		{"null", catalog(`{"prompt":null,"completion":"0"}`), 0},
		{"paid-id", strings.Replace(catalog(`{"prompt":"0","completion":"0"}`), "model:free", "model", 1), 0},
		{"router", strings.Replace(catalog(`{"prompt":"0","completion":"0"}`), "vendor/model:free", "openrouter/free", 1), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			models, err := Parse([]byte(tc.raw))
			if err != nil || len(models) != tc.want {
				t.Fatal(models, err)
			}
		})
	}
	for _, raw := range []string{`{}`, `{"data":null}`, `{"data":[],"data":[]}`, catalog(`{"prompt":"1","prompt":"0","completion":"0"}`), `{"data":[]`, strings.Repeat(" ", 4<<20) + "{}"} {
		if _, err := Parse([]byte(raw)); err == nil {
			t.Fatal("invalid catalog accepted")
		}
	}
}
func TestZeroPrice(t *testing.T) {
	for _, raw := range []string{`"0"`, `0`, `"0.000"`, `"0e-64"`} {
		if !Zero(json.RawMessage(raw)) {
			t.Fatal(raw)
		}
	}
	for _, raw := range []string{`null`, `""`, `"-0"`, `"-1"`, `"1e-9999"`, `"0/1"`, `"NaN"`, `"Inf"`, `" 0 "`, `"0.1"`} {
		if Zero(json.RawMessage(raw)) {
			t.Fatal(raw)
		}
	}
	for _, id := range []string{"openrouter/free", "vendor/model", "vendor/model:free:online", "vendor/model:free?plugin=x", "https://evil/free", "VENDOR/model:free"} {
		if FreeID(id) {
			t.Fatal(id)
		}
	}
}
func TestCatalogRequestNoCredentialOrGeneration(t *testing.T) {
	c := New()
	defer c.Close()
	c.http = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != Endpoint+"/models" || req.Method != "GET" || req.Header.Get("Authorization") != "" || req.Body != nil {
			t.Fatal("unsafe catalog request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(catalog(`{"prompt":"0","completion":"0"}`)))}, nil
	})}
	if err := c.Verify(context.Background(), "vendor/model:free"); err != nil {
		t.Fatal(err)
	}
	if err := c.Verify(context.Background(), "vendor/other:free"); err == nil {
		t.Fatal("missing model accepted")
	}
}
