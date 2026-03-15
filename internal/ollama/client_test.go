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

func TestChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chunks := []chatResponse{
			{Message: Message{Role: "assistant", Content: "こんにちは。"}, Done: false},
			{Message: Message{Role: "assistant", Content: "冒険を始めましょう。"}, Done: true},
		}
		encoder := json.NewEncoder(w)
		for _, chunk := range chunks {
			encoder.Encode(chunk)
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-model", 10*time.Second)

	var tokens []string
	result, err := c.ChatStream(context.Background(), []Message{
		{Role: "user", Content: "hi"},
	}, func(token string) {
		tokens = append(tokens, token)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "こんにちは。冒険を始めましょう。" {
		t.Errorf("expected full text, got '%s'", result)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(tokens))
	}
	if tokens[0] != "こんにちは。" {
		t.Errorf("expected first chunk 'こんにちは。', got '%s'", tokens[0])
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
