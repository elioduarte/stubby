package main

import (
	"net/http"

	"github.com/alexedwards/flow"
)

func (app *application) routes() http.Handler {
	mux := flow.New()

	mux.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowed)

	mux.Use(app.recoverPanic)
	mux.Use(app.skipIgnoredURL)
	// always force a fresh response from the server
	mux.Use(app.NoCacheMiddleware)

	mux.NotFound = http.HandlerFunc(app.forward)

	mux.HandleFunc("/_/record/:profile", app.recordProfile, "POST")
	mux.HandleFunc("/_/replay/:profile", app.replayProfile, "POST")
	mux.HandleFunc("/_/forward", app.enableForward, "POST")
	mux.HandleFunc("/_/status", app.responseStatus, "GET")

	return mux
}
