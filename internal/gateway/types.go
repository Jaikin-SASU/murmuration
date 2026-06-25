package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/Jaikin-SASU/murmuration/pkg/backend"
)

// completion couvre à la fois la réponse bufferisée (chat.completion) et les
// fragments streamés (chat.completion.chunk).
type completion struct {
	ID      string     `json:"id"`
	Object  string     `json:"object"`
	Created int64      `json:"created"`
	Model   string     `json:"model"`
	Choices []choice   `json:"choices"`
	Usage   *usageJSON `json:"usage,omitempty"`
}

type choice struct {
	Index        int                  `json:"index"`
	Message      *backend.ChatMessage `json:"message,omitempty"`
	Delta        *backend.ChatMessage `json:"delta,omitempty"`
	FinishReason *string              `json:"finish_reason"`
}

type usageJSON struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, errType, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"type": errType, "message": message},
	})
}
