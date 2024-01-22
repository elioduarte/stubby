package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"example.com/internal/version"
)

//go:embed config.json
var defaultConfig string

func main() {
	err := run()
	if err != nil {
		trace := string(debug.Stack())
		slog.Error(err.Error(), "trace", trace)
		os.Exit(1)
	}
}

func run() error {
	var cfg config
	cfg.ignoredPaths = []string{
		"/otlp",
		"/gw/otlp",
		"/api/alerts",
	}

	flag.StringVar(&cfg.baseURL, "base-url", "http://localhost:4444", "base URL for the application")
	flag.IntVar(&cfg.httpPort, "http-port", 4444, "port to listen on for HTTP requests")
	flag.StringVar(&cfg.stubDir, "stub-dir", "stubs", "directory to save the stub files")
	flag.Func("ignore-paths", "list of paths prefixes that should not be proxied (eg: /otlp/traces)", func(s string) error {
		cfg.ignoredPaths = strings.Split(s, ",")
		return nil
	})
	flag.Func("config-file", "path to the config file", func(s string) error {
		file, err := os.Open(s)
		if err != nil {
			return err
		}
		targets, err := parseConfig(file)
		if err != nil {
			return err
		}
		cfg.targets = targets
		return nil
	})

	showVersion := flag.Bool("version", false, "display version and exit")
	verbose := flag.Bool("verbose", false, "verbose")

	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\n", version.Get())
		return nil
	}

	lvl := slog.LevelInfo
	if *verbose {
		lvl = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))

	if cfg.targets == nil {
		targets, err := parseConfig(strings.NewReader(defaultConfig))
		if err != nil {
			return err
		}
		cfg.targets = targets
	}

	app := &application{
		config: cfg,
		logger: logger,
		proxy:  &httputil.ReverseProxy{},
	}
	app.proxy.Director = app.proxyDirector
	app.proxy.ErrorHandler = app.serverError

	return app.serveHTTP()
}

func parseConfig(r io.Reader) (*targets, error) {
	var t targets

	err := json.NewDecoder(r).Decode(&t)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	return &t, nil
}
