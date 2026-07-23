package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const healthPath = "/-/health"

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
	methodRE  = regexp.MustCompile(`^[!#$%&'*+\-.^_` + "`" + `|~0-9A-Za-z]+$`)
)

type endpointConfig struct {
	Request struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	} `json:"request"`
	Response struct {
		Status  int               `json:"status,omitempty"`
		Headers map[string]string `json:"headers,omitempty"`
		Body    json.RawMessage   `json:"body"`
	} `json:"response"`
	file string
}

type appConfig struct {
	configDir       string
	address         string
	readTimeout     time.Duration
	writeTimeout    time.Duration
	idleTimeout     time.Duration
	shutdownTimeout time.Duration
}

type mockHandler struct {
	routes  map[string]map[string]endpointConfig
	version string
}

func main() {
	flag.Parse()
	if flag.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "usage: go-mock-api-server [version|healthcheck]")
		os.Exit(2)
	}
	if flag.NArg() == 1 {
		switch flag.Arg(0) {
		case "version":
			fmt.Printf("go-mock-api-server %s (commit %s, built %s)\n", version, commit, buildDate)
			return
		case "healthcheck":
			if err := runHealthcheck(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		default:
			fmt.Fprintf(os.Stderr, "unknown command %q\n", flag.Arg(0))
			os.Exit(2)
		}
	}
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := configFromEnv()
	if err != nil {
		return err
	}
	endpoints, err := loadConfig(cfg.configDir)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr: cfg.address, Handler: newMockHandler(endpoints, version),
		ReadTimeout: cfg.readTimeout, WriteTimeout: cfg.writeTimeout, IdleTimeout: cfg.idleTimeout,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		slog.Info("mock API server started", "address", cfg.address, "config_dir", cfg.configDir, "routes", len(endpoints), "version", version)
		errCh <- server.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
		slog.Info("shutdown requested")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	}
}

func configFromEnv() (appConfig, error) {
	port := envOrDefault("SERVER_PORT", "8080")
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return appConfig{}, fmt.Errorf("SERVER_PORT must be an integer between 1 and 65535, got %q", port)
	}
	cfg := appConfig{configDir: envOrDefault("CONFIG_DIR", "config"), address: ":" + port}
	if strings.TrimSpace(cfg.configDir) == "" {
		return appConfig{}, errors.New("CONFIG_DIR must not be empty")
	}
	for _, item := range []struct {
		name     string
		value    *time.Duration
		fallback time.Duration
	}{
		{"READ_TIMEOUT", &cfg.readTimeout, 5 * time.Second},
		{"WRITE_TIMEOUT", &cfg.writeTimeout, 10 * time.Second},
		{"IDLE_TIMEOUT", &cfg.idleTimeout, 60 * time.Second},
		{"SHUTDOWN_TIMEOUT", &cfg.shutdownTimeout, 10 * time.Second},
	} {
		parsed, err := durationFromEnv(item.name, item.fallback)
		if err != nil {
			return appConfig{}, err
		}
		*item.value = parsed
	}
	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func durationFromEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := envOrDefault(name, fallback.String())
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive Go duration, got %q", name, value)
	}
	return parsed, nil
}

func loadConfig(dir string) ([]endpointConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read config directory %q: %w", dir, err)
	}
	var endpoints []endpointConfig
	seen := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		endpoint, err := decodeEndpoint(path)
		if err != nil {
			return nil, err
		}
		key := endpoint.Request.Method + " " + endpoint.Request.Path
		if previous, exists := seen[key]; exists {
			return nil, fmt.Errorf("duplicate route %s in %q and %q", key, previous, path)
		}
		seen[key] = path
		endpoints = append(endpoints, endpoint)
	}
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("config directory %q contains no JSON endpoint definitions", dir)
	}
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Request.Path == endpoints[j].Request.Path {
			return endpoints[i].Request.Method < endpoints[j].Request.Method
		}
		return endpoints[i].Request.Path < endpoints[j].Request.Path
	})
	return endpoints, nil
}

func decodeEndpoint(path string) (endpointConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return endpointConfig{}, fmt.Errorf("read %q: %w", path, err)
	}
	var endpoint endpointConfig
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&endpoint); err != nil {
		return endpointConfig{}, fmt.Errorf("decode %q: %w", path, err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return endpointConfig{}, fmt.Errorf("decode %q: %w", path, err)
	}
	endpoint.file = path
	endpoint.Request.Method = strings.ToUpper(strings.TrimSpace(endpoint.Request.Method))
	if err := validateEndpoint(endpoint); err != nil {
		return endpointConfig{}, fmt.Errorf("validate %q: %w", path, err)
	}
	if endpoint.Response.Status == 0 {
		endpoint.Response.Status = http.StatusOK
	}
	return endpoint, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("multiple JSON values are not allowed")
	}
	return err
}

func validateEndpoint(endpoint endpointConfig) error {
	method, path := endpoint.Request.Method, endpoint.Request.Path
	if !methodRE.MatchString(method) {
		return fmt.Errorf("request.method %q is not a valid HTTP method token", method)
	}
	if path == "" || path[0] != '/' {
		return fmt.Errorf("request.path %q must start with /", path)
	}
	if path != filepath.ToSlash(filepath.Clean(path)) || strings.ContainsAny(path, "?#") {
		return fmt.Errorf("request.path %q must be a clean URL path without query or fragment", path)
	}
	if path == healthPath {
		return fmt.Errorf("request.path %q is reserved for health checks", path)
	}
	if endpoint.Response.Status != 0 && (endpoint.Response.Status < 200 || endpoint.Response.Status > 599) {
		return errors.New("response.status must be between 200 and 599")
	}
	if len(endpoint.Response.Body) == 0 {
		return errors.New("response.body is required (use null for an empty JSON value)")
	}
	for name, value := range endpoint.Response.Headers {
		if !methodRE.MatchString(name) || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("response header %q contains an invalid name or value", name)
		}
		if strings.EqualFold(name, "Content-Length") || strings.EqualFold(name, "Transfer-Encoding") {
			return fmt.Errorf("response header %q is managed by the HTTP server and cannot be configured", name)
		}
	}
	return nil
}

func newMockHandler(endpoints []endpointConfig, appVersion string) http.Handler {
	routes := make(map[string]map[string]endpointConfig)
	for _, endpoint := range endpoints {
		if routes[endpoint.Request.Path] == nil {
			routes[endpoint.Request.Path] = make(map[string]endpointConfig)
		}
		routes[endpoint.Request.Path][endpoint.Request.Method] = endpoint
	}
	return &mockHandler{routes: routes, version: appVersion}
}

func (handler *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == healthPath {
		handler.serveHealth(w, r)
		return
	}
	methods, found := handler.routes[r.URL.Path]
	if !found {
		http.NotFound(w, r)
		return
	}
	endpoint, found := methods[r.Method]
	if !found && r.Method == http.MethodHead {
		endpoint, found = methods[http.MethodGet]
	}
	if !found {
		allowed := make([]string, 0, len(methods)+1)
		for method := range methods {
			allowed = append(allowed, method)
			if method == http.MethodGet {
				allowed = append(allowed, http.MethodHead)
			}
		}
		sort.Strings(allowed)
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	for name, value := range endpoint.Response.Headers {
		w.Header().Set(name, value)
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.WriteHeader(endpoint.Response.Status)
	if r.Method != http.MethodHead && endpoint.Response.Status != http.StatusNoContent && endpoint.Response.Status != http.StatusNotModified {
		if _, err := w.Write(endpoint.Response.Body); err != nil {
			slog.Warn("write response", "method", r.Method, "path", r.URL.Path, "error", err)
		}
	}
	slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", endpoint.Response.Status)
}

func (handler *mockHandler) serveHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = fmt.Fprintf(w, `{"status":"ok","version":%q}`+"\n", handler.version)
	}
}

func runHealthcheck() error {
	port := envOrDefault("SERVER_PORT", "8080")
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + port + healthPath)
	if err != nil {
		return fmt.Errorf("health check request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %s", response.Status)
	}
	return nil
}
