package backend

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Ollama pilote un serveur Ollama local (API REST, NDJSON en streaming).
// C'est le backend du pool de répliques (ADR-0003).
type Ollama struct {
	BaseURL string
	client  *http.Client
}

// NewOllama crée un backend Ollama pointant sur baseURL (ex. http://host:11434).
func NewOllama(baseURL string) *Ollama {
	return &Ollama{
		BaseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 0}, // pas de timeout global : le streaming peut durer
	}
}

// WithHTTPClient permet d'injecter un client (tests).
func (o *Ollama) WithHTTPClient(c *http.Client) *Ollama {
	o.client = c
	return o
}

func (o *Ollama) Name() string { return "ollama" }

// Available vérifie que le serveur répond.
func (o *Ollama) Available(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.BaseURL+"/api/version", nil)
	if err != nil {
		return err
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama indisponible: statut %d", resp.StatusCode)
	}
	return nil
}

// ListModels renvoie les modèles disponibles (GET /api/tags).
func (o *Ollama) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama /api/tags: statut %d", resp.StatusCode)
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Models))
	for _, m := range payload.Models {
		out = append(out, m.Name)
	}
	return out, nil
}

type ollamaChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  ollamaOptions `json:"options,omitempty"`
}

type ollamaOptions struct {
	Temperature float32 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaChatChunk struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done            bool `json:"done"`
	PromptEvalCount int  `json:"prompt_eval_count"`
	EvalCount       int  `json:"eval_count"`
}

// Generate appelle /api/chat en streaming NDJSON et relaie chaque fragment.
func (o *Ollama) Generate(ctx context.Context, req GenerateRequest, onChunk func(Chunk) error) (Usage, error) {
	body, err := json.Marshal(ollamaChatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
		Options:  ollamaOptions{Temperature: req.Temperature, NumPredict: req.MaxTokens},
	})
	if err != nil {
		return Usage{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Usage{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Usage{}, ErrModelNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return Usage{}, fmt.Errorf("ollama /api/chat: statut %d", resp.StatusCode)
	}

	var usage Usage
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var chunk ollamaChatChunk
		if err := json.Unmarshal(line, &chunk); err != nil {
			return usage, fmt.Errorf("ollama: fragment NDJSON invalide: %w", err)
		}
		if chunk.PromptEvalCount > 0 {
			usage.PromptTokens = chunk.PromptEvalCount
		}
		if chunk.EvalCount > 0 {
			usage.CompletionTokens = chunk.EvalCount
		}
		if chunk.Message.Content != "" || chunk.Done {
			if err := onChunk(Chunk{Content: chunk.Message.Content, Done: chunk.Done}); err != nil {
				return usage, err
			}
		}
		if chunk.Done {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return usage, err
	}
	return usage, nil
}

var _ Backend = (*Ollama)(nil)
