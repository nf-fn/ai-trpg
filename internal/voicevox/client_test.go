package voicevox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSynthesize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/audio_query":
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			text := r.URL.Query().Get("text")
			if text == "" {
				t.Error("expected text parameter")
			}
			speaker := r.URL.Query().Get("speaker")
			if speaker != "1" {
				t.Errorf("expected speaker 1, got %s", speaker)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"accent_phrases":[]}`))

		case "/synthesis":
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected application/json content-type")
			}
			w.Header().Set("Content-Type", "audio/wav")
			// Minimal WAV header for testing
			w.Write([]byte("RIFF\x00\x00\x00\x00WAVEfmt "))

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, 1)
	wav, err := c.Synthesize(context.Background(), "テスト")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wav) == 0 {
		t.Error("expected non-empty wav data")
	}
}

func TestSynthesizeServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	c := NewClient(server.URL, 1)
	_, err := c.Synthesize(context.Background(), "テスト")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
