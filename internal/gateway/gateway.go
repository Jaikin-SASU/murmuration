// Package gateway expose le contrat public stable : une API OpenAI-compatible
// (ADR-0006). Les clients existants fonctionnent sans modification. Le gateway
// ne décide pas où router : il délègue à un Router (implémenté par le serveur
// au-dessus du scheduler).
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Jaikin-SASU/murmuration/pkg/backend"
)

// Router choisit un backend pour un modèle donné et liste les modèles servis.
type Router interface {
	Route(ctx context.Context, model string) (backend.Backend, string, error)
	Models(ctx context.Context) []string
}

// Gateway sert l'API publique.
type Gateway struct {
	router Router
	now    func() time.Time
}

// New crée un gateway.
func New(r Router) *Gateway {
	return &Gateway{router: r, now: time.Now}
}

// Handler renvoie le routeur HTTP du plan public.
func (g *Gateway) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", g.handleModels)
	mux.HandleFunc("POST /v1/chat/completions", g.handleChat)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

func (g *Gateway) handleModels(w http.ResponseWriter, r *http.Request) {
	ids := g.router.Models(r.Context())
	type model struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	}
	out := struct {
		Object string  `json:"object"`
		Data   []model `json:"data"`
	}{Object: "list", Data: make([]model, 0, len(ids))}
	for _, id := range ids {
		out.Data = append(out.Data, model{ID: id, Object: "model", OwnedBy: "murmuration"})
	}
	writeJSON(w, http.StatusOK, out)
}

type chatRequest struct {
	Model       string                `json:"model"`
	Messages    []backend.ChatMessage `json:"messages"`
	Stream      bool                  `json:"stream"`
	Temperature float32               `json:"temperature"`
	MaxTokens   int                   `json:"max_tokens"`
}

func (g *Gateway) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "corps JSON invalide")
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "le champ 'model' est requis")
		return
	}
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request_error", "au moins un message est requis")
		return
	}

	be, _, err := g.router.Route(r.Context(), req.Model)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "no_capacity", "aucun node disponible: "+err.Error())
		return
	}

	genReq := backend.GenerateRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if req.Stream {
		g.streamChat(w, r, be, genReq)
		return
	}
	g.bufferChat(w, r, be, genReq)
}

func (g *Gateway) bufferChat(w http.ResponseWriter, r *http.Request, be backend.Backend, req backend.GenerateRequest) {
	var sb strings.Builder
	usage, err := be.Generate(r.Context(), req, func(c backend.Chunk) error {
		sb.WriteString(c.Content)
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	stop := "stop"
	resp := completion{
		ID:      g.id(),
		Object:  "chat.completion",
		Created: g.now().Unix(),
		Model:   req.Model,
		Choices: []choice{{
			Index:        0,
			Message:      &backend.ChatMessage{Role: "assistant", Content: sb.String()},
			FinishReason: &stop,
		}},
		Usage: &usageJSON{
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:      usage.PromptTokens + usage.CompletionTokens,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (g *Gateway) streamChat(w http.ResponseWriter, r *http.Request, be backend.Backend, req backend.GenerateRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "server_error", "streaming non supporté")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	id := g.id()
	created := g.now().Unix()

	emit := func(delta *backend.ChatMessage, finish *string) {
		ch := completion{
			ID:      id,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   req.Model,
			Choices: []choice{{Index: 0, Delta: delta, FinishReason: finish}},
		}
		b, _ := json.Marshal(ch)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	// Premier chunk : le rôle, comme OpenAI.
	emit(&backend.ChatMessage{Role: "assistant"}, nil)

	_, err := be.Generate(r.Context(), req, func(c backend.Chunk) error {
		if c.Content != "" {
			emit(&backend.ChatMessage{Content: c.Content}, nil)
		}
		return nil
	})
	if err != nil {
		// En SSE, on signale l'erreur dans le flux puis on clôt.
		b, _ := json.Marshal(map[string]any{"error": map[string]string{"message": err.Error()}})
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
		return
	}

	stop := "stop"
	emit(&backend.ChatMessage{}, &stop)
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func (g *Gateway) id() string {
	return fmt.Sprintf("chatcmpl-%d", g.now().UnixNano())
}
