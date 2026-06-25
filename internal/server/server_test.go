package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Jaikin-SASU/murmuration/pkg/backend"
	"github.com/Jaikin-SASU/murmuration/pkg/capability"
	"github.com/Jaikin-SASU/murmuration/pkg/registry"
	"github.com/Jaikin-SASU/murmuration/pkg/scheduler"
)

func newTestServer(auth Authenticator) *Server {
	return New(Options{
		Registry:  registry.New(10 * time.Second),
		Scheduler: scheduler.New(scheduler.DefaultPolicy()),
		Auth:      auth,
		// Le factory ignore le node et renvoie un backend déterministe.
		Factory: func(capability.Node, string) (backend.Backend, error) {
			return backend.NewFake([]string{"llama3"}, "ok"), nil
		},
	})
}

func registerNode(t *testing.T, srv *httptest.Server, n capability.Node) {
	t.Helper()
	b, _ := json.Marshal(n)
	resp, err := http.Post(srv.URL+"/internal/v1/heartbeat", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat statut = %d", resp.StatusCode)
	}
}

func freeNode(id string) capability.Node {
	return capability.Node{
		ID:       id,
		Presence: capability.PresenceFree,
		Caps: capability.Capabilities{
			GPUs:         []capability.GPUInfo{{Vendor: capability.VendorNVIDIA, VRAMTotalMB: 24000, VRAMFreeMB: 20000}},
			Backends:     []string{"ollama"},
			LoadedModels: []string{"llama3"},
			Endpoints:    map[string]string{"ollama": "http://127.0.0.1:11434"},
		},
	}
}

func TestHeartbeatThenRoute(t *testing.T) {
	s := newTestServer(AllowAll{})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	registerNode(t, srv, freeNode("n1"))

	// Models doit refléter le modèle chargé.
	if got := s.Models(context.Background()); len(got) != 1 || got[0] != "llama3" {
		t.Fatalf("Models = %v", got)
	}

	// Une requête chat doit être routée et produire une réponse.
	body := `{"model":"llama3","messages":[{"role":"user","content":"hi"}]}`
	resp, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat statut = %d", resp.StatusCode)
	}
}

func TestDefaultBackendFactory(t *testing.T) {
	n := capability.Node{Caps: capability.Capabilities{Endpoints: map[string]string{"ollama": "http://x:11434"}}}
	if _, err := DefaultBackendFactory(n, "ollama"); err != nil {
		t.Fatalf("factory devrait réussir: %v", err)
	}
	if _, err := DefaultBackendFactory(capability.Node{}, "ollama"); err == nil {
		t.Fatal("factory devrait échouer sans endpoint")
	}
}

func TestListNodesEndpoint(t *testing.T) {
	s := newTestServer(AllowAll{})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	registerNode(t, srv, freeNode("n1"))

	resp, err := http.Get(srv.URL + "/internal/v1/nodes")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var nodes []capability.Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].ID != "n1" {
		t.Fatalf("nodes = %+v", nodes)
	}
}

func TestRouteNoNodes(t *testing.T) {
	s := newTestServer(AllowAll{})
	_, _, err := s.Route(context.Background(), "llama3")
	if err == nil {
		t.Fatal("attendu une erreur sans node enrôlé")
	}
}

func TestAuthRequired(t *testing.T) {
	s := newTestServer(StaticToken{Token: "secret"})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	registerNode(t, srv, freeNode("n1"))

	body := `{"model":"llama3","messages":[{"role":"user","content":"hi"}]}`

	// Sans token → 401.
	resp, _ := http.Post(srv.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("sans token: statut = %d, attendu 401", resp.StatusCode)
	}

	// Avec token → OK.
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("avec token: statut = %d, attendu 200", resp2.StatusCode)
	}
}
