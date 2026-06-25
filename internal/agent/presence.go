package agent

import (
	"time"

	"github.com/Jaikin-SASU/murmuration/pkg/capability"
)

// PresenceDetector évalue l'état de présence local du poste. C'est le garde-fou
// qui garantit la priorité à l'utilisateur assis devant (ADR-0005).
type PresenceDetector interface {
	Detect() capability.PresenceState
}

// StaticDetector renvoie toujours le même état. Utile pour un poste dédié
// (toujours Free) ou pour les tests.
type StaticDetector struct{ State capability.PresenceState }

func (d StaticDetector) Detect() capability.PresenceState { return d.State }

// IdleBased dérive la présence du temps écoulé depuis le dernier input et de
// l'état de la session. Les sondes OS (input réel, session) sont injectées :
// le câblage natif Windows/macOS viendra remplir IdleInput/SessionOpen.
type IdleBased struct {
	// ActiveThreshold : en deçà de cette inactivité, l'utilisateur est Actif.
	ActiveThreshold time.Duration
	// IdleInput : durée depuis le dernier input clavier/souris.
	IdleInput func() time.Duration
	// SessionOpen : vrai si une session utilisateur est ouverte.
	SessionOpen func() bool
}

// Detect applique la politique à 3 niveaux.
func (d IdleBased) Detect() capability.PresenceState {
	if d.SessionOpen != nil && !d.SessionOpen() {
		return capability.PresenceFree
	}
	if d.IdleInput != nil && d.IdleInput() < d.ActiveThreshold {
		return capability.PresenceActive
	}
	return capability.PresenceIdle
}
