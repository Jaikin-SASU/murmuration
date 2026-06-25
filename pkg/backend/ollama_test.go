package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("chemin inattendu %q", r.URL.Path)
		}
		w.Write([]byte(`{"models":[{"name":"llama3:8b"},{"name":"mistral"}]}`))
	}))
	defer srv.Close()

	models, err := NewOllama(srv.URL).ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0] != "llama3:8b" {
		t.Fatalf("modèles inattendus: %v", models)
	}
}

func TestOllamaGenerateStreaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("chemin inattendu %q", r.URL.Path)
		}
		// NDJSON : trois fragments puis done.
		w.Write([]byte(`{"message":{"content":"Bon"},"prompt_eval_count":5}` + "\n"))
		w.Write([]byte(`{"message":{"content":"jour"}}` + "\n"))
		w.Write([]byte(`{"message":{"content":""},"done":true,"eval_count":2}` + "\n"))
	}))
	defer srv.Close()

	var got strings.Builder
	var sawDone bool
	usage, err := NewOllama(srv.URL).Generate(context.Background(),
		GenerateRequest{Model: "llama3", Messages: []ChatMessage{{Role: "user", Content: "salut"}}},
		func(c Chunk) error {
			got.WriteString(c.Content)
			if c.Done {
				sawDone = true
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "Bonjour" {
		t.Fatalf("contenu agrégé = %q, attendu Bonjour", got.String())
	}
	if !sawDone {
		t.Error("le fragment final Done n'a pas été relayé")
	}
	if usage.PromptTokens != 5 || usage.CompletionTokens != 2 {
		t.Fatalf("usage inattendu: %+v", usage)
	}
}

func TestOllamaModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := NewOllama(srv.URL).Generate(context.Background(),
		GenerateRequest{Model: "absent"}, func(Chunk) error { return nil })
	if err != ErrModelNotFound {
		t.Fatalf("attendu ErrModelNotFound, obtenu %v", err)
	}
}

func TestOllamaAvailable(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/version" {
			t.Errorf("chemin %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	if err := NewOllama(ok.URL).Available(context.Background()); err != nil {
		t.Fatalf("Available devrait réussir: %v", err)
	}

	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer down.Close()
	if err := NewOllama(down.URL).Available(context.Background()); err == nil {
		t.Fatal("Available devrait échouer sur 500")
	}
}

func TestFakeBackend(t *testing.T) {
	f := NewFake([]string{"m1"}, "a", "b")
	var out strings.Builder
	usage, err := f.Generate(context.Background(), GenerateRequest{Model: "m1"}, func(c Chunk) error {
		out.WriteString(c.Content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "ab" {
		t.Fatalf("sortie fake = %q", out.String())
	}
	if usage.CompletionTokens != 2 {
		t.Fatalf("usage fake = %+v", usage)
	}
}
