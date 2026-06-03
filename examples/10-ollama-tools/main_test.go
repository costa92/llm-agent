package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestProviderSupportsTools exercises the REAL provider strategy (no network):
// only the llama3.1 / qwen2.5-coder / qwen3-coder families are tool-capable.
func TestProviderSupportsTools(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"llama3.1:8b", true},
		{"qwen2.5-coder:7b", true},
		{"qwen3-coder:latest", true},
		{"llama3:latest", false}, // llama3, not llama3.1
		{"gemma4:latest", false}, // server says "tools", provider disagrees
		{"deepseek-r1:7b", false},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := providerSupportsTools(defaultHost, tt.model); got != tt.want {
				t.Errorf("providerSupportsTools(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", defaultHost},
		{"localhost:11434", "http://localhost:11434"},
		{"http://example:1234/", "http://example:1234"},
		{"https://host", "https://host"},
	}
	for _, tt := range tests {
		if got := normalizeHost(tt.in); got != tt.want {
			t.Errorf("normalizeHost(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// tagsServer serves a fake GET /api/tags returning the given JSON body.
func tagsServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

func TestPickToolModel_AutoSelectsToolsCapable(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "") // force auto-detect
	// First model is not tool-capable and must be skipped for the llama3.1 one.
	srv := tagsServer(t, `{"models":[
		{"name":"llama3:latest"},
		{"name":"llama3.1:8b"}
	]}`)
	defer srv.Close()

	got, err := pickToolModel(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("pickToolModel returned error: %v", err)
	}
	if got != "llama3.1:8b" {
		t.Errorf("pickToolModel = %q, want the tools-capable model %q", got, "llama3.1:8b")
	}
}

func TestPickToolModel_EnvOverrideWins(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "qwen2.5-coder")
	srv := httptest.NewServer(http.NotFoundHandler()) // would fail if probed
	defer srv.Close()

	got, err := pickToolModel(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("pickToolModel returned error: %v", err)
	}
	if got != "qwen2.5-coder" {
		t.Errorf("pickToolModel = %q, want %q", got, "qwen2.5-coder")
	}
}

func TestPickToolModel_NoToolsCapableModel(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "")
	srv := tagsServer(t, `{"models":[
		{"name":"llama3:latest"},
		{"name":"gemma4:latest"}
	]}`)
	defer srv.Close()

	_, err := pickToolModel(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected an error when no tools-capable model is available")
	}
	if !strings.Contains(err.Error(), "support tool-calling") {
		t.Errorf("error = %q, want it to mention 'support tool-calling'", err)
	}
}

func TestPickToolModel_ServerUnreachable(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "")
	srv := tagsServer(t, `{}`)
	url := srv.URL
	srv.Close() // close so the connection is refused

	_, err := pickToolModel(context.Background(), url)
	if err == nil {
		t.Fatal("expected an error when the server is unreachable")
	}
	if !strings.Contains(err.Error(), "cannot reach Ollama") {
		t.Errorf("error = %q, want it to mention 'cannot reach Ollama'", err)
	}
}

func TestListModels_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := listModels(context.Background(), srv.URL); err == nil {
		t.Fatal("expected an error on non-200 response")
	}
}
