package response

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Wrapper struct {
	http.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func (rw *Wrapper) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (rw *Wrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *Wrapper) Header() http.Header {
	return rw.ResponseWriter.Header()
}

func (rw *Wrapper) Text() string {
	return rw.body.String()
}

func (rw *Wrapper) StatusCode() int {
	return rw.statusCode
}

func (rw *Wrapper) Body() (interface{}, error) {
	if rw.body.Len() == 0 {
		return "", nil
	}

	if IsPlainText(rw.Header()) {
		return rw.body.String(), nil
	}

	var reader io.Reader = bytes.NewReader(rw.body.Bytes())

	if IsGzip(rw.Header()) {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip body: %w", err)
		}
		reader = gzipReader
	}

	if rw.ContentType() != "" && !IsJSON(rw.Header()) {
		return nil, fmt.Errorf("content-type is not application/json")
	}

	var result interface{}

	err := json.NewDecoder(reader).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal body to JSON: %w", err)
	}

	return result, nil
}

func (rw *Wrapper) ContentEncoding() any {
	return rw.Header().Get(HeaderContentEncoding)
}

func (rw *Wrapper) ContentType() any {
	return rw.Header().Get(HeaderContentType)
}
