package response

import (
	"net/http"
	"strings"
)

const (
	HeaderContentEncoding = "Content-Encoding"
	HeaderContentType     = "Content-Type"
)

func isGzip(h http.Header) bool {
	return h.Get(HeaderContentEncoding) == "gzip"
}

func isJSON(h http.Header) bool {
	return strings.HasPrefix(h.Get(HeaderContentType), "application/json")
}

func isPlainText(h http.Header) bool {
	return strings.HasPrefix(h.Get(HeaderContentType), "text/plain")
}
