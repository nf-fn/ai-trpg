package gm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/nf-fn/ai-trpg/internal/ollama"
)

type mockOllamaClient struct {
	response string
	err      error
	received []ollama.Message
}

func (m *mockOllamaClient) Chat(_ context.Context, messages []ollama.Message) (string, error) {
	m.received = messages
	return m.response, m.err
}

func (m *mockOllamaClient) ChatStream(_ context.Context, messages []ollama.Message, onChunk func(token string)) (string, error) {
	m.received = messages
	if m.err != nil {
		return "", m.err
	}
	// Simulate streaming by splitting response into runes
	for _, r := range m.response {
		if onChunk != nil {
			onChunk(string(r))
		}
	}
	return m.response, nil
}

func TestPlayerAction(t *testing.T) {
	mock := &mockOllamaClient{response: "あなたは暗い洞窟の入り口に立っている。"}
	engine := NewEngine(mock)

	rule := &Rule{Name: "test", Content: "テスト用ルール"}
	scenario := &Scenario{Name: "test", Content: "テスト用シナリオ"}
	engine.StartSession(rule, scenario)

	result, err := engine.PlayerAction(context.Background(), "洞窟に入る")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "あなたは暗い洞窟の入り口に立っている。" {
		t.Errorf("unexpected result: %s", result)
	}

	// Verify messages sent to Ollama
	if len(mock.received) != 2 {
		t.Fatalf("expected 2 messages (system + user), got %d", len(mock.received))
	}
	if mock.received[0].Role != "system" {
		t.Errorf("expected system role, got %s", mock.received[0].Role)
	}
	if mock.received[1].Role != "user" {
		t.Errorf("expected user role, got %s", mock.received[1].Role)
	}
	if mock.received[1].Content != "洞窟に入る" {
		t.Errorf("expected user content '洞窟に入る', got '%s'", mock.received[1].Content)
	}
}

func TestPlayerActionError(t *testing.T) {
	mock := &mockOllamaClient{err: fmt.Errorf("connection refused")}
	engine := NewEngine(mock)
	engine.StartSession(nil, nil)

	_, err := engine.PlayerAction(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// History should not contain the failed message
	if len(engine.history) != 1 {
		t.Errorf("expected 1 message (system only), got %d", len(engine.history))
	}
}

func TestConversationHistory(t *testing.T) {
	mock := &mockOllamaClient{}
	engine := NewEngine(mock)
	engine.StartSession(nil, nil)

	mock.response = "応答1"
	engine.PlayerAction(context.Background(), "行動1")

	mock.response = "応答2"
	engine.PlayerAction(context.Background(), "行動2")

	// system + user1 + assistant1 + user2
	if len(mock.received) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(mock.received))
	}
}

func TestPlayerActionStream(t *testing.T) {
	mock := &mockOllamaClient{response: "洞窟は暗い。"}
	engine := NewEngine(mock)
	engine.StartSession(nil, nil)

	var chunks []string
	result, err := engine.PlayerActionStream(context.Background(), "洞窟に入る", func(token string) {
		chunks = append(chunks, token)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "洞窟は暗い。" {
		t.Errorf("unexpected result: %s", result)
	}
	if len(chunks) == 0 {
		t.Error("expected chunks, got none")
	}

	// History should include system + user + assistant
	if len(engine.history) != 3 {
		t.Errorf("expected 3 messages, got %d", len(engine.history))
	}
}

func TestPlayerActionStreamError(t *testing.T) {
	mock := &mockOllamaClient{err: fmt.Errorf("timeout")}
	engine := NewEngine(mock)
	engine.StartSession(nil, nil)

	_, err := engine.PlayerActionStream(context.Background(), "test", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(engine.history) != 1 {
		t.Errorf("expected 1 message (system only), got %d", len(engine.history))
	}
}

func TestLoadRules(t *testing.T) {
	dir := t.TempDir()
	content := `name: テストルール
description: テスト用
content: |
  1. ダイスは2D6を使用
  2. 判定値以上で成功
`
	if err := os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := LoadRules(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Name != "テストルール" {
		t.Errorf("unexpected name: %s", rules[0].Name)
	}
}

func TestLoadScenarios(t *testing.T) {
	dir := t.TempDir()
	content := `name: テストシナリオ
description: テスト用
content: |
  舞台は中世ファンタジーの世界。
`
	if err := os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	scenarios, err := LoadScenarios(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].Name != "テストシナリオ" {
		t.Errorf("unexpected name: %s", scenarios[0].Name)
	}
}
