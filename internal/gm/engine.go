package gm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nf-fn/ai-trpg/internal/ollama"
	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Content     string `yaml:"content"`
}

type Scenario struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Content     string `yaml:"content"`
}

type Engine struct {
	ollamaClient ollama.Client
	history      []ollama.Message
	maxHistory   int
	rule         *Rule
	scenario     *Scenario
}

func NewEngine(ollamaClient ollama.Client, maxHistory int) *Engine {
	return &Engine{
		ollamaClient: ollamaClient,
		maxHistory:   maxHistory,
	}
}

func (e *Engine) StartSession(rule *Rule, scenario *Scenario) {
	e.rule = rule
	e.scenario = scenario
	e.history = []ollama.Message{
		{Role: "system", Content: e.buildSystemPrompt()},
	}
}

func (e *Engine) PlayerAction(ctx context.Context, text string) (string, error) {
	e.history = append(e.history, ollama.Message{
		Role:    "user",
		Content: text,
	})

	response, err := e.ollamaClient.Chat(ctx, e.history)
	if err != nil {
		// Remove the failed user message
		e.history = e.history[:len(e.history)-1]
		return "", fmt.Errorf("ollama chat: %w", err)
	}

	e.history = append(e.history, ollama.Message{
		Role:    "assistant",
		Content: response,
	})

	e.pruneHistory()

	return response, nil
}

func (e *Engine) PlayerActionStream(ctx context.Context, text string, onChunk func(token string)) (string, error) {
	e.history = append(e.history, ollama.Message{
		Role:    "user",
		Content: text,
	})

	response, err := e.ollamaClient.ChatStream(ctx, e.history, onChunk)
	if err != nil {
		e.history = e.history[:len(e.history)-1]
		return "", fmt.Errorf("ollama chat: %w", err)
	}

	e.history = append(e.history, ollama.Message{
		Role:    "assistant",
		Content: response,
	})

	e.pruneHistory()

	return response, nil
}

// pruneHistory keeps system prompt (index 0) + last maxHistory messages.
func (e *Engine) pruneHistory() {
	if e.maxHistory <= 0 {
		return
	}
	// history[0] is system, rest are user/assistant messages
	msgs := len(e.history) - 1
	if msgs <= e.maxHistory {
		return
	}
	e.history = append(e.history[:1], e.history[len(e.history)-e.maxHistory:]...)
}

func (e *Engine) buildSystemPrompt() string {
	var sb strings.Builder

	sb.WriteString("あなたはTRPGのゲームマスター（GM）です。以下のルールとシナリオに従ってゲームを進行してください。\n\n")

	if e.rule != nil {
		sb.WriteString("--- ルール ---\n")
		sb.WriteString(e.rule.Content)
		sb.WriteString("\n\n")
	}

	if e.scenario != nil {
		sb.WriteString("--- シナリオ ---\n")
		sb.WriteString(e.scenario.Content)
		sb.WriteString("\n\n")
	}

	sb.WriteString("--- 指示 ---\n")
	sb.WriteString("- プレイヤーの行動に応じて物語を進行してください\n")
	sb.WriteString("- 判定が必要な場合はダイスロールを指示してください（例: 「2D6で判定してください」）\n")
	sb.WriteString("- NPCのセリフは「」で囲んでください\n")
	sb.WriteString("- 場面描写は情景が浮かぶように詳細に行ってください\n")
	sb.WriteString("- プレイヤーが楽しめるようにテンポよく進行してください\n")

	return sb.String()
}

func LoadRules(dir string) ([]Rule, error) {
	return loadYAMLFiles[Rule](dir)
}

func LoadScenarios(dir string) ([]Scenario, error) {
	return loadYAMLFiles[Scenario](dir)
}

func loadYAMLFiles[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var items []T
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		var item T
		if err := yaml.Unmarshal(data, &item); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		items = append(items, item)
	}

	return items, nil
}
