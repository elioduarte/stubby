package main

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"example.com/internal/stubby"

	"example.com/internal/response"
	"github.com/alexedwards/flow"
)

func (app *application) statusHandler(w http.ResponseWriter, r *http.Request) {
	app.statusLock.RLock()
	data := struct {
		Profile string   `json:"profile"`
		Status  string   `json:"status"`
		Targets *targets `json:"targets"`
	}{
		Profile: app.profile,
		Status:  app.status.String(),
		Targets: app.config.targets,
	}
	app.statusLock.RUnlock()

	err := response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) recordHandler(w http.ResponseWriter, r *http.Request) {
	profile, err := app.currentProfile(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	app.changeStatus(Recording, profile)

	app.statusHandler(w, r)
}

func (app *application) replayHandler(w http.ResponseWriter, r *http.Request) {
	profile, err := app.currentProfile(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	app.changeStatus(Replaying, profile)

	err = app.loadProfile(profile)
	if err != nil {
		app.unprocessableEntity(w, r, err)
		return
	}

	app.statusHandler(w, r)
}

func (app *application) forwardHandler(w http.ResponseWriter, r *http.Request) {
	app.changeStatus(Forwarding, "")

	app.statusHandler(w, r)
}

func (app *application) forward(w http.ResponseWriter, r *http.Request) {
	status := app.currentStatus()

	app.rewrite(r)

	if app.replay(status, w, r) {
		return
	}

	rw := &response.Wrapper{ResponseWriter: w}
	app.proxy.ServeHTTP(rw, r)

	app.logger.Info("responseForwarded",
		"http.method", r.Method,
		"http.path", r.URL.Path,
		"http.query", r.URL.RawQuery,
		"http.status_code", rw.StatusCode(),
		"http.content_encoding", rw.ContentEncoding(),
		"http.content_type", rw.ContentType(),
	)

	app.backgroundTask(r, func() error {
		return app.record(status, rw, r)
	})
}

func (app *application) rewrite(r *http.Request) {
	var found bool

	for _, targetURL := range app.config.targets.Prefixes {
		if targetURL.Matches(r) {
			targetURL.Rewrite(r)
			found = true
			break
		}
	}

	if !found {
		app.config.targets.Default.Rewrite(r)
	}

	app.logger.Debug("requestModified",
		"http.method", r.Method,
		"http.host", r.URL.Host,
		"http.path", r.URL.Path,
		"http.scheme", r.URL.Scheme,
		"http.query", r.URL.RawQuery,
	)
}

func (app *application) record(status Status, rw *response.Wrapper, r *http.Request) error {
	app.logger.Debug("recordingResponse",
		"http.method", r.Method,
		"http.path", r.URL.Path,
		"http.scheme", r.URL.Scheme,
		"http.query", r.URL.RawQuery,
		"http.status_code", rw.StatusCode(),
		"http.content_encoding", rw.ContentEncoding(),
		"http.content_type", rw.ContentType(),
	)

	if status != Recording {
		app.logger.Debug("recordDropped",
			"http.method", r.Method,
			"http.path", r.URL.Path,
			"http.scheme", r.URL.Scheme,
			"http.query", r.URL.RawQuery,
			"http.status_code", rw.StatusCode(),
			"http.content_encoding", rw.ContentEncoding(),
			"http.content_type", rw.ContentType(),
		)
		return nil
	}

	body, err := rw.Body()
	if err != nil {
		return fmt.Errorf("failed to response body: %w", err)
	}

	record := stubby.Record{
		Profile: app.profile,
		Request: stubby.Request{
			Host:     r.URL.Host,
			Pathname: r.URL.Path,
			Method:   r.Method,
			Query:    response.QueryToJSON(r.URL.Query()),
		},
		Response: stubby.Response{
			StatusCode: rw.StatusCode(),
			Body:       body,
		},
	}

	app.recordsLock.Lock()
	app.records = append(app.records, &record)
	app.recordsLock.Unlock()
	app.wg.Add(1)

	app.logger.Debug("responseRecorded",
		"http.method", r.Method,
		"http.path", r.URL.Path,
		"http.scheme", r.URL.Scheme,
		"http.query", r.URL.RawQuery,
		"http.status_code", rw.StatusCode(),
		"http.content_encoding", rw.ContentEncoding(),
		"http.content_type", rw.ContentType(),
	)

	return nil
}

func (app *application) replay(status Status, w http.ResponseWriter, r *http.Request) bool {
	if status != Replaying {
		return false
	}

	record, ok := app.matcher.Match(r)
	if !ok {
		return false
	}

	err := response.JSON(w, record.Response.StatusCode, record.Response.Body)
	if err != nil {
		app.serverError(w, r, fmt.Errorf("failed to replay response: %w", err))
		return true
	}

	app.logger.Info("responseReplayed",
		"http.method", r.Method,
		"http.path", r.URL.Path,
		"http.query", r.URL.RawQuery,
		"http.status_code", record.Response.StatusCode,
		"stub.file", record.Filepath(),
	)

	return true
}

func (app *application) currentProfile(r *http.Request) (string, error) {
	profile := flow.Param(r.Context(), "profile")
	if profile == "" {
		return "", errors.New("empty profile")
	}

	return strings.ToLower(profile), nil
}

func (app *application) loadProfile(profile string) error {
	app.logger.Debug("loadingProfile", "profile", profile)

	profilePath := filepath.Join(app.config.stubDir, profile)
	matcher, err := stubby.NewMatcher(profilePath)
	if err != nil {
		return fmt.Errorf("failed to create stub matcher %s: %w", profilePath, err)
	}

	app.matcher = matcher

	return nil
}

func (app *application) changeStatus(status Status, profile string) {
	app.logger.Info("changeStatus", "new", status, "profile", profile)

	app.statusLock.Lock()
	defer app.statusLock.Unlock()

	app.status = status
	app.profile = strings.ToLower(profile)
}

func (app *application) currentStatus() Status {
	app.statusLock.RLock()
	defer app.statusLock.RUnlock()

	return app.status
}
