// Package registry conserve l'état vivant des nodes du parc. Un node dont le
// heartbeat a expiré (poste éteint, déconnecté, ou utilisateur revenu) sort
// automatiquement des résultats — c'est la base de la robustesse (ADR-0005).
package registry

import (
	"sync"
	"time"

	"github.com/Jaikin-SASU/murmuration/pkg/capability"
)

// Registry est sûr pour un usage concurrent.
type Registry struct {
	mu    sync.RWMutex
	ttl   time.Duration
	nodes map[string]capability.Node
	now   func() time.Time
}

// New crée un registry où un node est considéré vivant tant que son dernier
// heartbeat date de moins de ttl.
func New(ttl time.Duration) *Registry {
	return &Registry{
		ttl:   ttl,
		nodes: make(map[string]capability.Node),
		now:   time.Now,
	}
}

// SetClock remplace l'horloge (tests).
func (r *Registry) SetClock(now func() time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.now = now
}

// Upsert insère ou met à jour un node et estampille son heartbeat à maintenant.
// Le node est cloné : l'appelant ne peut pas muter l'état interne après coup.
func (r *Registry) Upsert(node capability.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored := node.Clone()
	stored.LastHeartbeat = r.now()
	r.nodes[node.ID] = stored
}

// Get renvoie un clone du node et s'il est vivant.
func (r *Registry) Get(id string) (capability.Node, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.nodes[id]
	if !ok || !r.aliveLocked(n) {
		return capability.Node{}, false
	}
	return n.Clone(), true
}

// List renvoie les clones de tous les nodes vivants.
func (r *Registry) List() []capability.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]capability.Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		if r.aliveLocked(n) {
			out = append(out, n.Clone())
		}
	}
	return out
}

// Remove retire explicitement un node (désenrôlement).
func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.nodes, id)
}

// Prune supprime définitivement les nodes expirés et renvoie le nombre retiré.
func (r *Registry) Prune() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	removed := 0
	for id, n := range r.nodes {
		if !r.aliveLocked(n) {
			delete(r.nodes, id)
			removed++
		}
	}
	return removed
}

func (r *Registry) aliveLocked(n capability.Node) bool {
	return r.now().Sub(n.LastHeartbeat) <= r.ttl
}
