package transport

import (
	"net/http"
	"portway/internal/httpjson"
)

func Decode(w http.ResponseWriter, r *http.Request, out any) error {
	return httpjson.Decode(w, r, out)
}
