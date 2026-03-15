package server

import (
	"github.com/nf-fn/ai-trpg/internal/gm"
	"github.com/nf-fn/ai-trpg/internal/ollama"
	"github.com/nf-fn/ai-trpg/internal/voicevox"
)

type Server struct {
	config         *Config
	ollamaClient   ollama.Client
	voicevoxClient voicevox.Client
	gmEngine       *gm.Engine
}

func New(cfg *Config, oc ollama.Client, vc voicevox.Client, ge *gm.Engine) *Server {
	return &Server{
		config:         cfg,
		ollamaClient:   oc,
		voicevoxClient: vc,
		gmEngine:       ge,
	}
}
