package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/nf-fn/ai-trpg/internal/gm"
)

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir(s.config.Paths.Web)))
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/rules", s.handleRules)
	mux.HandleFunc("/api/scenarios", s.handleScenarios)

	return mux
}

type listItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	rules, err := gm.LoadRules(s.config.Paths.Rules)
	if err != nil {
		log.Printf("load rules: %v", err)
		http.Error(w, "failed to load rules", http.StatusInternalServerError)
		return
	}

	items := make([]listItem, len(rules))
	for i, rule := range rules {
		items[i] = listItem{Name: rule.Name, Description: rule.Description}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) handleScenarios(w http.ResponseWriter, r *http.Request) {
	scenarios, err := gm.LoadScenarios(s.config.Paths.Scenarios)
	if err != nil {
		log.Printf("load scenarios: %v", err)
		http.Error(w, "failed to load scenarios", http.StatusInternalServerError)
		return
	}

	items := make([]listItem, len(scenarios))
	for i, sc := range scenarios {
		items[i] = listItem{Name: sc.Name, Description: sc.Description}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
