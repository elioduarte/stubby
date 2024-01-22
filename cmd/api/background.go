package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"example.com/internal/stubby"
)

func (app *application) backgroundTask(r *http.Request, fn func() error) {
	app.wg.Add(1)

	go func() {
		defer app.wg.Done()

		defer func() {
			err := recover()
			if err != nil {
				app.reportServerError(r, fmt.Errorf("%s", err))
			}
		}()

		err := fn()
		if err != nil {
			app.reportServerError(r, err)
		}
	}()
}

func (app *application) watchRecords(done chan struct{}) {
	app.logger.Debug("startWatchingRecords")
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-done:
			app.logger.Debug("stopWatchingRecords")
			return
		case <-ticker.C:
			app.checkRecords()
		}
	}
}

func (app *application) checkRecords() {
	app.recordsLock.Lock()
	records := make([]*stubby.Record, len(app.records))
	copy(records, app.records)
	app.records = nil
	app.recordsLock.Unlock()

	for _, record := range records {
		app.writeFile(record)
		app.wg.Done()
	}
}

func (app *application) writeFile(record *stubby.Record) {
	fullPath := app.fullPath(record)

	err := stubby.WriteToFile(fullPath, record)
	if err != nil {
		app.logger.Debug("writeFileFailed",
			"stub.path", fullPath,
			"stub.file", record.Filepath(),
			"http.method", record.Request.Method,
			"http.path", record.Request.Pathname,
			"http.query", record.Request.Query,
			"http.status_code", record.Response.StatusCode,
		)
	}

	app.logger.Debug("writeFileSucceed",
		"stub.path", record.Filepath(),
		"stub.file", record.Filepath(),
		"http.method", record.Request.Method,
		"http.path", record.Request.Pathname,
		"http.query", record.Request.Query,
		"http.status_code", record.Response.StatusCode,
	)
}

func (app *application) fullPath(record *stubby.Record) string {
	return filepath.Join(app.config.stubDir, record.Filepath())
}
