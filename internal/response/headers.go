package response

import (
	"net/http"
	"strings"
)

const (
	HeaderContentEncoding = "Content-Encoding"
	HeaderContentType     = "Content-Type"
)

func IsGzip(h http.Header) bool {
	return h.Get(HeaderContentEncoding) == "gzip"
}

func IsJSON(h http.Header) bool {
	return strings.HasPrefix(h.Get(HeaderContentType), "application/json")
}

func IsPlainText(h http.Header) bool {
	return strings.HasPrefix(h.Get(HeaderContentType), "text/plain")
}
