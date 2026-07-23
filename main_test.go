package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "get.json", `{
        "request":{"method":"get","path":"/v1/users"},
        "response":{"status":201,"headers":{"X-Test":"yes"},"body":{"ok":true}}
    }`)
	writeConfig(t, dir, "post.json", `{
        "request":{"method":"POST","path":"/v1/users"},
        "response":{"body":null}
    }`)

	endpoints, err := loadConfig(dir)
	if err != nil {
		t.Fatalf("loadConfig returned an error: %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("got %d endpoints, want 2", len(endpoints))
	}
	if endpoints[0].Request.Method != http.MethodGet || endpoints[0].Response.Status != http.StatusCreated {
		t.Fatalf("endpoint was not normalized/defaulted correctly: %#v", endpoints[0])
	}
}

func TestBundledConfig(t *testing.T) {
	endpoints, err := loadConfig("config")
	if err != nil {
		t.Fatalf("bundled config must be valid: %v", err)
	}
	if len(endpoints) != 4 {
		t.Fatalf("got %d bundled endpoints, want 4", len(endpoints))
	}

	var foundCreate bool
	for _, endpoint := range endpoints {
		if endpoint.Request.Method == http.MethodPost && endpoint.Request.Path == "/v1/user/create" {
			foundCreate = true
			if endpoint.Response.Status != http.StatusCreated {
				t.Fatalf("create status = %d, want %d", endpoint.Response.Status, http.StatusCreated)
			}
			if endpoint.Response.Headers["Location"] != "/v1/user/3" {
				t.Fatalf("unexpected create Location header: %q", endpoint.Response.Headers["Location"])
			}
		}
	}
	if !foundCreate {
		t.Fatal("bundled create route was not found")
	}
}

func TestLoadConfigRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		wantErr string
	}{
		{"empty directory", nil, "contains no JSON"},
		{"unknown field", map[string]string{"a.json": `{"request":{"method":"GET","path":"/a","extra":true},"response":{"body":{}}}`}, "unknown field"},
		{"missing method", map[string]string{"a.json": `{"request":{"path":"/a"},"response":{"body":{}}}`}, "request.method"},
		{"unclean path", map[string]string{"a.json": `{"request":{"method":"GET","path":"/a/../b"},"response":{"body":{}}}`}, "clean URL path"},
		{"reserved health", map[string]string{"a.json": `{"request":{"method":"GET","path":"/-/health"},"response":{"body":{}}}`}, "reserved"},
		{"missing body", map[string]string{"a.json": `{"request":{"method":"GET","path":"/a"},"response":{}}`}, "response.body"},
		{"invalid status", map[string]string{"a.json": `{"request":{"method":"GET","path":"/a"},"response":{"status":199,"body":{}}}`}, "response.status"},
		{"managed header", map[string]string{"a.json": `{"request":{"method":"GET","path":"/a"},"response":{"headers":{"Content-Length":"9"},"body":{}}}`}, "managed by the HTTP server"},
		{"duplicate route", map[string]string{
			"a.json": `{"request":{"method":"GET","path":"/a"},"response":{"body":{}}}`,
			"b.json": `{"request":{"method":"get","path":"/a"},"response":{"body":{}}}`,
		}, "duplicate route"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range test.files {
				writeConfig(t, dir, name, content)
			}
			_, err := loadConfig(dir)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("got error %v, want one containing %q", err, test.wantErr)
			}
		})
	}
}

func TestMockHandler(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "route.json", `{
        "request":{"method":"GET","path":"/v1/item"},
        "response":{"status":202,"headers":{"X-Mock":"true"},"body":{"id":1}}
    }`)
	writeConfig(t, dir, "no-content.json", `{
        "request":{"method":"DELETE","path":"/v1/item"},
        "response":{"status":204,"body":null}
    }`)
	endpoints, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	handler := newMockHandler(endpoints, "v1.2.3")

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
		wantAllow  string
	}{
		{"configured route", http.MethodGet, "/v1/item", 202, `{"id":1}`, ""},
		{"no-content route", http.MethodDelete, "/v1/item", 204, "", ""},
		{"head uses get", http.MethodHead, "/v1/item", 202, "", ""},
		{"wrong method", http.MethodPost, "/v1/item", 405, "method not allowed", "DELETE, GET, HEAD"},
		{"exact path", http.MethodGet, "/v1/item/child", 404, "404 page not found", ""},
		{"health", http.MethodGet, healthPath, 200, `{"status":"ok","version":"v1.2.3"}`, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", response.Body.String(), test.wantBody)
			}
			if response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf("Allow = %q, want %q", response.Header().Get("Allow"), test.wantAllow)
			}
		})
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("CONFIG_DIR", "/tmp/mocks")
	t.Setenv("READ_TIMEOUT", "2s")
	cfg, err := configFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.address != ":9090" || cfg.configDir != "/tmp/mocks" || cfg.readTimeout != 2*time.Second {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestConfigFromEnvRejectsInvalidValues(t *testing.T) {
	t.Setenv("SERVER_PORT", "70000")
	if _, err := configFromEnv(); err == nil {
		t.Fatal("expected invalid port error")
	}

	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("READ_TIMEOUT", "never")
	if _, err := configFromEnv(); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestRunHealthcheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != healthPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, healthPath)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	t.Setenv("SERVER_PORT", port)
	if err := runHealthcheck(); err != nil {
		t.Fatal(err)
	}
}

func writeConfig(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
