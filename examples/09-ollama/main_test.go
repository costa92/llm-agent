package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatCapable(t *testing.T) {
	tests := []struct {
		name  string
		model ollamaModel
		want  bool
	}{
		{"completion capability", ollamaModel{Name: "gemma4:latest", Capabilities: []string{"completion", "tools"}}, true},
		{"embedding only", ollamaModel{Name: "nomic-embed-text:latest", Capabilities: []string{"embedding"}}, false},
		{"no caps, chat name", ollamaModel{Name: "llama3:8b"}, true},
		{"no caps, embed name", ollamaModel{Name: "mxbai-embed-large:latest"}, false},
		{"no caps, m3e name", ollamaModel{Name: "m3e:large-f16"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.model.chatCapable(); got != tt.want {
				t.Errorf("chatCapable() = %v, want %v", got, tt.want)
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

func TestPickModel_AutoSelectsFirstChatModel(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "") // force auto-detect
	srv := tagsServer(t, `{"models":[
		{"name":"nomic-embed-text:latest","capabilities":["embedding"]},
		{"name":"gemma4:latest","capabilities":["completion","tools"]}
	]}`)
	defer srv.Close()

	got, err := pickModel(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("pickModel returned error: %v", err)
	}
	if got != "gemma4:latest" {
		t.Errorf("pickModel = %q, want the chat model %q", got, "gemma4:latest")
	}
}

func TestPickModel_EnvOverrideWins(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "qwen2.5")
	// Point at a server that would 404 /api/tags — proving the env value is
	// returned without ever probing the catalog.
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	got, err := pickModel(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("pickModel returned error: %v", err)
	}
	if got != "qwen2.5" {
		t.Errorf("pickModel = %q, want %q", got, "qwen2.5")
	}
}

func TestPickModel_ServerUnreachable(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "")
	srv := tagsServer(t, `{}`)
	url := srv.URL
	srv.Close() // close so the connection is refused

	_, err := pickModel(context.Background(), url)
	if err == nil {
		t.Fatal("expected an error when the server is unreachable")
	}
	if !strings.Contains(err.Error(), "cannot reach Ollama") {
		t.Errorf("error = %q, want it to mention 'cannot reach Ollama'", err)
	}
}

func TestPickModel_NoChatModel(t *testing.T) {
	t.Setenv("OLLAMA_MODEL", "")
	srv := tagsServer(t, `{"models":[{"name":"nomic-embed-text:latest","capabilities":["embedding"]}]}`)
	defer srv.Close()

	_, err := pickModel(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected an error when no chat model is available")
	}
	if !strings.Contains(err.Error(), "no chat-capable model") {
		t.Errorf("error = %q, want it to mention 'no chat-capable model'", err)
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
