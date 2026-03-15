package server

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Ollama   OllamaConfig   `yaml:"ollama"`
	Voicevox VoicevoxConfig `yaml:"voicevox"`
	Whisper  WhisperConfig  `yaml:"whisper"`
	Server   ServerConfig   `yaml:"server"`
	Paths    PathsConfig    `yaml:"paths"`
}

type PathsConfig struct {
	Web       string `yaml:"web"`
	Rules     string `yaml:"rules"`
	Scenarios string `yaml:"scenarios"`
}

type OllamaConfig struct {
	URL     string        `yaml:"url"`
	Model   string        `yaml:"model"`
	Timeout time.Duration `yaml:"timeout"`
}

type VoicevoxConfig struct {
	URL     string        `yaml:"url"`
	Speaker int           `yaml:"speaker"`
	Timeout time.Duration `yaml:"timeout"`
}

type WhisperConfig struct {
	Model string `yaml:"model"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Ollama: OllamaConfig{
			URL:     "http://localhost:11434",
			Model:   "gemma2",
			Timeout: 60 * time.Second,
		},
		Voicevox: VoicevoxConfig{
			URL:     "http://localhost:50021",
			Speaker: 1,
			Timeout: 30 * time.Second,
		},
		Whisper: WhisperConfig{
			Model: "base",
		},
		Server: ServerConfig{
			Port: 8080,
		},
		Paths: PathsConfig{
			Web:       "web",
			Rules:     "rules",
			Scenarios: "scenarios",
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
