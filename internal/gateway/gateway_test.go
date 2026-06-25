package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jaikin-SASU/murmuration/pkg/backend"
)

type fakeRouter struct {
	be     backend.Backend
	models []string
	err    error
}

func (f fakeRouter) Route(context.Context, string) (backend.Backend, string, error) {
	if f.err != nil {
		return nil, "", f.err
	}
	return f.be, "node-1", nil
}
func (f fakeRouter) Models(context.Context) []string { return f.models }

func TestChatCompletionBuffered(t *testing.T) {
	g := New(fakeRouter{be: backend.NewFake([]string{"llama3"}, "Bon", "jour")})
	srv := httptest.NewServer(g.Handler())
	defer srv.Close()

	body := `{"model":"llama3","messages":[{"role":"user","content":"salut"}]}`
	resp, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statut = %d", resp.StatusCode)
	}

	var out completion
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Choices) != 1 || out.Choices[0].Message == nil {
		t.Fatalf("réponse inattendue: %+v", out)
	}
	if out.Choices[0].Message.Content != "Bonjour" {
		t.Fatalf("contenu = %q, attendu Bonjour", out.Choices[0].Message.Content)
	}
	if out.Object != "chat.completion" {
		t.Errorf("object = %q", out.Object)
	}
}

func TestChatCompletionStreaming(t *testing.T) {
	g := New(fakeRouter{be: backend.NewFake([]string{"llama3"}, "A", "B")})
	srv := httptest.NewServer(g.Handler())
	defer srv.Close()

	body := `{"model":"llama3","stream":true,"messages":[{"role":"user","content":"x"}]}`
	resp, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q", ct)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	stream := string(raw)
	if !strings.Contains(stream, "chat.completion.chunk") {
		t.Error("flux sans chunk")
	}
	if !strings.Contains(stream, "data: [DONE]") {
		t.Error("flux sans marqueur [DONE]")
	}
}

func TestChatNoCapacity(t *testing.T) {
	g := New(fakeRouter{err: context.DeadlineExceeded})
	srv := httptest.NewServer(g.Handler())
	defer srv.Close()

	body := `{"model":"llama3","messages":[{"role":"user","content":"x"}]}`
	resp, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("statut = %d, attendu 503", resp.StatusCode)
	}
}

func TestChatValidation(t *testing.T) {
	g := New(fakeRouter{be: backend.NewFake(nil)})
	srv := httptest.NewServer(g.Handler())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"messages":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("statut = %d, attendu 400", resp.StatusCode)
	}
}

func TestModels(t *testing.T) {
	g := New(fakeRouter{models: []string{"llama3", "mistral"}})
	srv := httptest.NewServer(g.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Data) != 2 {
		t.Fatalf("attendu 2 modèles, obtenu %d", len(out.Data))
	}
}
