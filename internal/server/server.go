// Package server câble le control plane : registry (état du parc), scheduler
// (décision), gateway (API publique) et les endpoints internes par lesquels les
// agents publient leur télémétrie.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/Jaikin-SASU/murmuration/internal/gateway"
	"github.com/Jaikin-SASU/murmuration/pkg/backend"
	"github.com/Jaikin-SASU/murmuration/pkg/capability"
	"github.com/Jaikin-SASU/murmuration/pkg/registry"
	"github.com/Jaikin-SASU/murmuration/pkg/scheduler"
)

// BackendFactory construit le Backend joignable pour un node donné. Injectable
// pour les tests.
type BackendFactory func(n capability.Node, backendName string) (backend.Backend, error)

// DefaultBackendFactory construit un backend Ollama à partir de l'endpoint
// publié par le node.
func DefaultBackendFactory(n capability.Node, backendName string) (backend.Backend, error) {
	url := n.Caps.Endpoints[backendName]
	if url == "" {
		return nil, errNoEndpoint
	}
	return backend.NewOllama(url), nil
}

type cfgError struct{ msg string }

func (e cfgError) Error() string { return e.msg }

var errNoEndpoint = cfgError{"le node ne publie pas d'endpoint pour ce backend"}

// Server est le control plane.
type Server struct {
	reg     *registry.Registry
	sched   *scheduler.Scheduler
	factory BackendFactory
	auth    Authenticator
	gw      *gateway.Gateway
}

// Options configure le serveur.
type Options struct {
	Registry  *registry.Registry
	Scheduler *scheduler.Scheduler
	Factory   BackendFactory
	Auth      Authenticator
}

// New crée le serveur. Les champs nil reçoivent des valeurs par défaut.
func New(opts Options) *Server {
	s := &Server{
		reg:     opts.Registry,
		sched:   opts.Scheduler,
		factory: opts.Factory,
		auth:    opts.Auth,
	}
	if s.reg == nil {
		s.reg = registry.New(defaultTTL)
	}
	if s.sched == nil {
		s.sched = scheduler.New(scheduler.DefaultPolicy())
	}
	if s.factory == nil {
		s.factory = DefaultBackendFactory
	}
	if s.auth == nil {
		s.auth = AllowAll{}
	}
	s.gw = gateway.New(s)
	return s
}

// Route implémente gateway.Router : choisit un node via le scheduler et
// construit son backend.
func (s *Server) Route(ctx context.Context, model string) (backend.Backend, string, error) {
	d, err := s.sched.Pick(s.reg.List(), scheduler.Request{Model: model})
	if err != nil {
		return nil, "", err
	}
	be, err := s.factory(d.Node, "ollama")
	if err != nil {
		return nil, "", err
	}
	return be, d.Node.ID, nil
}

// Models implémente gateway.Router : union des modèles chargés sur le parc.
func (s *Server) Models(ctx context.Context) []string {
	set := make(map[string]struct{})
	for _, n := range s.reg.List() {
		for _, m := range n.Caps.LoadedModels {
			set[m] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for m := range set {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

// Handler renvoie le routeur HTTP complet (public + interne).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Plan public (API OpenAI), protégé par l'authentificateur.
	mux.Handle("/v1/", s.withAuth(s.gw.Handler()))
	mux.Handle("/healthz", s.gw.Handler())

	// Plan de contrôle : les agents publient leur télémétrie.
	mux.HandleFunc("POST /internal/v1/heartbeat", s.handleHeartbeat)
	mux.HandleFunc("GET /internal/v1/nodes", s.handleListNodes)

	return mux
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := s.auth.Authenticate(r); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"type": "unauthorized", "message": err.Error()},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var node capability.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, "JSON invalide", http.StatusBadRequest)
		return
	}
	if node.ID == "" {
		http.Error(w, "champ 'id' requis", http.StatusBadRequest)
		return
	}
	s.reg.Upsert(node)
	// v1 : pas de directive serveur→agent (ADR-0008, l'agent poll au heartbeat).
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.reg.List())
}
