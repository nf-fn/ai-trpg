package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/nf-fn/ai-trpg/internal/gm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type incomingMessage struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Scenario string `json:"scenario,omitempty"`
}

type outgoingMessage struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Index int    `json:"index,omitempty"`
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade: %v", err)
		return
	}
	defer conn.Close()

	engine := gm.NewEngine(s.ollamaClient, s.config.GM.MaxHistory)
	wsMu := &sync.Mutex{} // Protect concurrent writes

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("websocket read: %v", err)
			}
			return
		}

		var msg incomingMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("unmarshal message: %v", err)
			continue
		}

		switch msg.Type {
		case "start":
			s.handleStart(conn, wsMu, engine, msg)
		case "message":
			s.handleMessage(conn, wsMu, engine, msg)
		default:
			log.Printf("unknown message type: %s", msg.Type)
		}
	}
}

func (s *Server) handleStart(conn *websocket.Conn, wsMu *sync.Mutex, engine *gm.Engine, msg incomingMessage) {
	rules, err := gm.LoadRules(s.config.Paths.Rules)
	if err != nil {
		sendError(conn, wsMu, "ルールの読み込みに失敗しました")
		return
	}

	scenarios, err := gm.LoadScenarios(s.config.Paths.Scenarios)
	if err != nil {
		sendError(conn, wsMu, "シナリオの読み込みに失敗しました")
		return
	}

	var selectedRule *gm.Rule
	for i, r := range rules {
		if r.Name == msg.Rule {
			selectedRule = &rules[i]
			break
		}
	}

	var selectedScenario *gm.Scenario
	for i, sc := range scenarios {
		if sc.Name == msg.Scenario {
			selectedScenario = &scenarios[i]
			break
		}
	}

	engine.StartSession(selectedRule, selectedScenario)

	s.sendStreamingResponse(conn, wsMu, engine, "セッションを開始してください。舞台の説明と、プレイヤーキャラクターの作成を案内してください。")
}

func (s *Server) handleMessage(conn *websocket.Conn, wsMu *sync.Mutex, engine *gm.Engine, msg incomingMessage) {
	s.sendStreamingResponse(conn, wsMu, engine, msg.Text)
}

// sendStreamingResponse streams GM response sentence by sentence with audio.
// Flow: Ollama streams tokens → buffer until 。→ synthesize sentence → send chunk+WAV → repeat → send done.
func (s *Server) sendStreamingResponse(conn *websocket.Conn, wsMu *sync.Mutex, engine *gm.Engine, text string) {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Ollama.Timeout)
	defer cancel()

	var (
		buf   strings.Builder
		index int
	)

	// Flush accumulated buffer as a chunk with audio
	flushSentence := func(sentence string) {
		index++
		// Send text chunk
		wsMu.Lock()
		conn.WriteJSON(outgoingMessage{Type: "chunk", Text: sentence, Index: index})
		wsMu.Unlock()

		// Synthesize audio
		ttsCtx, ttsCancel := context.WithTimeout(context.Background(), s.config.Voicevox.Timeout)
		defer ttsCancel()
		wav, err := s.voicevoxClient.Synthesize(ttsCtx, sentence)
		if err != nil {
			log.Printf("voicevox synthesize chunk %d: %v", index, err)
			// Continue without audio — text was already sent
			return
		}

		wsMu.Lock()
		conn.WriteMessage(websocket.BinaryMessage, wav)
		wsMu.Unlock()
	}

	_, err := engine.PlayerActionStream(ctx, text, func(token string) {
		buf.WriteString(token)

		// Check for sentence boundary (。)
		content := buf.String()
		for {
			idx := strings.Index(content, "。")
			if idx < 0 {
				break
			}
			sentence := content[:idx+len("。")]
			flushSentence(sentence)
			content = content[idx+len("。"):]
		}
		buf.Reset()
		buf.WriteString(content)
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			sendError(conn, wsMu, "GMの応答がタイムアウトしました。再度お試しください")
		} else {
			sendError(conn, wsMu, "GMの応答生成に失敗しました: "+err.Error())
		}
		return
	}

	// Flush remaining buffer (text without trailing 。)
	remaining := strings.TrimSpace(buf.String())
	if remaining != "" {
		flushSentence(remaining)
	}

	// Signal completion
	wsMu.Lock()
	conn.WriteJSON(outgoingMessage{Type: "done"})
	wsMu.Unlock()
}

func sendError(conn *websocket.Conn, wsMu *sync.Mutex, text string) {
	wsMu.Lock()
	defer wsMu.Unlock()
	resp := outgoingMessage{Type: "error", Text: text}
	conn.WriteJSON(resp)
}
