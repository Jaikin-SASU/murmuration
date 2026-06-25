// Package backend abstrait le moteur d'inférence d'un node. C'est l'interface
// qui permet le 10x sans réécriture (ADR-0006) : Ollama et llama.cpp RPC
// aujourd'hui, vLLM/MLX/futurs demain, sans toucher au scheduler.
package backend

import (
	"context"
	"errors"
)

// ErrModelNotFound : le modèle demandé n'est pas disponible sur ce backend.
var ErrModelNotFound = errors.New("backend: modèle introuvable")

// ChatMessage est un message de conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerateRequest décrit une demande de génération.
type GenerateRequest struct {
	Model       string
	Messages    []ChatMessage
	Temperature float32
	MaxTokens   int
}

// Chunk est un fragment de réponse streamé.
type Chunk struct {
	Content string
	Done    bool
}

// Usage compte les tokens consommés.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// Backend est implémenté par chaque moteur. Generate streame les fragments via
// onChunk ; renvoyer une erreur depuis onChunk interrompt proprement le flux.
type Backend interface {
	Name() string
	Available(ctx context.Context) error
	ListModels(ctx context.Context) ([]string, error)
	Generate(ctx context.Context, req GenerateRequest, onChunk func(Chunk) error) (Usage, error)
}
