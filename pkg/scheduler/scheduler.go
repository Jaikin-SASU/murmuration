// Package scheduler décide quel node sert une requête. Règles (ADR-0005) :
//   - priorité absolue à l'utilisateur : un node Actif est inéligible ;
//   - gros GPU d'abord : le tier GPU pèse fort dans le score ;
//   - on préfère un node qui a DÉJÀ le modèle chargé (sinon scale-up) ;
//   - réseau faible : on garde une requête sur un seul node (pas d'éclatement ici).
package scheduler

import (
	"errors"
	"sort"

	"github.com/Jaikin-SASU/murmuration/pkg/capability"
)

// ErrNoCapacity : aucun node éligible pour cette requête.
var ErrNoCapacity = errors.New("scheduler: aucun node éligible")

// Policy paramètre le scoring. Valeurs par défaut via DefaultPolicy().
type Policy struct {
	// VRAMWeight : poids du Go de VRAM libre dans le score.
	VRAMWeight float64
	// IdleFactor : facteur de présence appliqué aux nodes Idle (< 1 → plafonnés).
	IdleFactor float64
	// LoadPenalty : pénalité par requête en vol + élément de file.
	LoadPenalty float64
	// MinFreeVRAMMB : VRAM libre minimale pour être éligible (0 = pas de plancher).
	MinFreeVRAMMB int64
}

// DefaultPolicy renvoie une politique raisonnable pour un parc de bureau.
func DefaultPolicy() Policy {
	return Policy{
		VRAMWeight:    1.0,
		IdleFactor:    0.5,
		LoadPenalty:   2.0,
		MinFreeVRAMMB: 0,
	}
}

// Scheduler applique une Policy.
type Scheduler struct {
	policy Policy
}

// New crée un scheduler.
func New(p Policy) *Scheduler { return &Scheduler{policy: p} }

// Request décrit ce qu'on cherche à placer.
type Request struct {
	Model   string
	Backend string // backend requis, ex. "ollama" (défaut si vide)
}

// Decision : le node choisi, son score, et s'il faut d'abord charger le modèle.
type Decision struct {
	Node      capability.Node
	Score     float64
	NeedsLoad bool
}

// gpuTier traduit le meilleur GPU d'un node en multiplicateur de score.
// Gros GPU dédié ≫ GPU moyen ≫ Apple Silicon ≫ CPU-only.
func gpuTier(c capability.Capabilities) float64 {
	g, ok := c.BestGPU()
	if !ok {
		return 0.2 // CPU-only
	}
	switch g.Vendor {
	case capability.VendorNVIDIA, capability.VendorAMD:
		switch {
		case g.VRAMTotalMB >= 16000:
			return 4.0
		case g.VRAMTotalMB >= 8000:
			return 2.5
		default:
			return 1.5
		}
	case capability.VendorApple:
		return 1.0
	default:
		return 0.2
	}
}

func presenceFactor(p capability.PresenceState, pol Policy) (float64, bool) {
	switch p {
	case capability.PresenceFree:
		return 1.0, true
	case capability.PresenceIdle:
		return pol.IdleFactor, true
	default: // Active → inéligible
		return 0, false
	}
}

// Eligible indique si un node peut recevoir la requête (présence, backend, VRAM).
func (s *Scheduler) Eligible(n capability.Node, req Request) bool {
	if _, ok := presenceFactor(n.Presence, s.policy); !ok {
		return false
	}
	backend := req.Backend
	if backend == "" {
		backend = "ollama"
	}
	if !n.Caps.SupportsBackend(backend) {
		return false
	}
	if s.policy.MinFreeVRAMMB > 0 && n.Caps.FreeVRAMMB() < s.policy.MinFreeVRAMMB {
		return false
	}
	return true
}

// Score calcule le score d'un node pour une requête. Suppose Eligible == true.
func (s *Scheduler) Score(n capability.Node, req Request) float64 {
	pf, _ := presenceFactor(n.Presence, s.policy)
	freeVRAMGB := float64(n.Caps.FreeVRAMMB()) / 1024.0
	base := s.policy.VRAMWeight * freeVRAMGB * gpuTier(n.Caps) * pf
	load := float64(n.InflightRequests+n.QueueDepth) * s.policy.LoadPenalty
	return base - load
}

// Pick choisit le meilleur node. On privilégie ceux qui ont déjà le modèle
// chargé ; à défaut, on retourne le meilleur candidat à chauffer (NeedsLoad).
func (s *Scheduler) Pick(nodes []capability.Node, req Request) (Decision, error) {
	var withModel, withoutModel []Decision
	for _, n := range nodes {
		if !s.Eligible(n, req) {
			continue
		}
		d := Decision{Node: n, Score: s.Score(n, req)}
		if n.Caps.HasModel(req.Model) {
			withModel = append(withModel, d)
		} else {
			d.NeedsLoad = true
			withoutModel = append(withoutModel, d)
		}
	}

	if best, ok := bestOf(withModel); ok {
		return best, nil
	}
	if best, ok := bestOf(withoutModel); ok {
		return best, nil
	}
	return Decision{}, ErrNoCapacity
}

// bestOf renvoie la décision au score le plus haut (tri stable par ID pour
// un résultat déterministe en cas d'égalité).
func bestOf(ds []Decision) (Decision, bool) {
	if len(ds) == 0 {
		return Decision{}, false
	}
	sort.SliceStable(ds, func(i, j int) bool {
		if ds[i].Score == ds[j].Score {
			return ds[i].Node.ID < ds[j].Node.ID
		}
		return ds[i].Score > ds[j].Score
	})
	return ds[0], true
}
