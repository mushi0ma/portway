package transport

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStrictJSON(t *testing.T) {
	for _, tc := range []struct {
		body string
		ok   bool
	}{
		{`{"name":"ok","config":{"value":1}}`, true},
		{`{"name":"x","name":"y"}`, false},
		{`{"Name":"x"}`, false},
		{`{"name":"x","unknown":true}`, false},
		{`{"name":"x","config":{"unknown":1}}`, false},
		{`{"config":{"value":1,"value":2}}`, false},
		{`null`, false}, {`[]`, false}, {`{} {}`, false}, {`{"config":{"value":1.5}}`, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			var v struct {
				Name   string `json:"name"`
				Config struct {
					Value int `json:"value"`
				} `json:"config"`
			}
			err := Decode(httptest.NewRecorder(), r, &v)
			if (err == nil) != tc.ok {
				t.Fatalf("ok=%v, err=%v", tc.ok, err)
			}
		})
	}
}
