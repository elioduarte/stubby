package main

import (
	"fmt"
	"net/http"
	"strings"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) skipIgnoredURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, prefix := range app.config.ignoredPaths {
			if strings.HasPrefix(r.URL.Path, prefix) {
				w.WriteHeader(http.StatusOK)
				app.logger.Debug("pathSkipped", "http.path", r.URL.Path)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) NoCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cacheHeaders := []string{
			"If-Modified-Since",
			"If-Unmodified-Since",
			"If-Match",
			"If-None-Match",
			"Cache-Control",
			"Pragma",
		}

		for _, header := range cacheHeaders {
			r.Header.Del(header)
		}

		next.ServeHTTP(w, r)
	})
}
