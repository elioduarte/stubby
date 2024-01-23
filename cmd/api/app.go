package main

import (
	"log/slog"
	"net/http/httputil"
	"sync"

	"example.com/internal/stubby"
)

type Status int

const (
	Forwarding Status = iota
	Replaying
	Recording
)

func (s Status) String() string {
	return []string{"Forwarding", "Replaying", "Recording"}[s]
}

type targets struct {
	Default  stubby.Target   `json:"default"`
	Prefixes []stubby.Target `json:"prefixes"`
}

type config struct {
	baseURL      string
	httpPort     int
	stubDir      string
	ignoredPaths []string
	targets      *targets
}

type application struct {
	config      config
	logger      *slog.Logger
	wg          sync.WaitGroup
	proxy       *httputil.ReverseProxy
	status      Status
	statusLock  sync.RWMutex
	profile     string
	recordsLock sync.Mutex
	records     []*stubby.Record
	matcher     *stubby.Matcher
}
