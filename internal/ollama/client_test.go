package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected /api/chat, got %s", r.URL.Path)
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("expected model test-model, got %s", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(req.Messages))
		}

		w.Header().Set("Content-Type", "application/json")
		// Simulate streaming response
		chunks := []chatResponse{
			{Message: Message{Role: "assistant", Content: "Hello"}, Done: false},
			{Message: Message{Role: "assistant", Content: " world"}, Done: true},
		}
		encoder := json.NewEncoder(w)
		for _, chunk := range chunks {
			encoder.Encode(chunk)
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-model", 10*time.Second)
	result, err := c.Chat(context.Background(), []Message{
		{Role: "user", Content: "hi"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello world" {
		t.Errorf("expected 'Hello world', got '%s'", result)
	}
}

func TestChatServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-model", 10*time.Second)
	_, err := c.Chat(context.Background(), []Message{
		{Role: "user", Content: "hi"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
