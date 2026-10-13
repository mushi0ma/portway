package httpjson

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"portway/internal/problem"
)

func Decode(w http.ResponseWriter, r *http.Request, out any) error {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		return problem.New(415, "UNSUPPORTED_MEDIA_TYPE", "Content-Type must be application/json")
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		return problem.New(413, "REQUEST_TOO_LARGE", "Request exceeds 1 MiB")
	}
	return DecodeObject(b, out)
}

// DecodeObject validates bounded JSON from either HTTP or a model tool proposal.
// It rejects duplicate, unknown and noncanonical fields before typed decoding.
func DecodeObject(b []byte, out any) error {
	if len(b) > 1<<20 {
		return problem.New(413, "REQUEST_TOO_LARGE", "JSON exceeds 1 MiB")
	}
	if !utf8.Valid(b) || len(bytes.TrimSpace(b)) == 0 || bytes.TrimSpace(b)[0] != '{' {
		return problem.Invalid("Expected a UTF-8 JSON object")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err := uniqueJSON(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return problem.Invalid("Expected one JSON value")
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return problem.Invalid("Unknown fields or invalid JSON value")
	}
	return nil
}
func uniqueJSON(d *json.Decoder, depth int) error {
	if depth > 32 {
		return problem.Invalid("JSON nesting exceeds limit")
	}
	t, err := d.Token()
	if err != nil {
		return problem.Invalid("Malformed JSON")
	}
	if delim, ok := t.(json.Delim); ok {
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return problem.Invalid("Malformed JSON")
				}
				key, ok := k.(string)
				if !ok || seen[key] || key != strings.ToLower(key) {
					return problem.Invalid("Duplicate or noncanonical JSON field")
				}
				seen[key] = true
				if err = uniqueJSON(d, depth+1); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return problem.Invalid("Malformed JSON")
			}
		case '[':
			for d.More() {
				if err := uniqueJSON(d, depth+1); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return problem.Invalid("Malformed JSON")
			}
		default:
			return problem.Invalid("Malformed JSON")
		}
	}
	return nil
}
