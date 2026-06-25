// Package agent est le superviseur local qui tourne sur chaque poste. Il publie
// présence + télémétrie au control plane à intervalle régulier (ADR-0008). Il ne
// fait pas l'inférence : il décrit ce que le node sait faire.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Jaikin-SASU/murmuration/pkg/capability"
)

// Agent publie l'état d'un node.
type Agent struct {
	ServerURL string
	Base      capability.Node // ID, Hostname, Caps de base
	Detector  PresenceDetector
	Interval  time.Duration

	// LoadedModels, si fourni, rafraîchit la liste des modèles chargés à chaque
	// heartbeat (typiquement via le backend local).
	LoadedModels func(ctx context.Context) []string

	// GPUProbe, si fourni, redétecte les GPU à chaque heartbeat (VRAM libre à
	// jour). Voir DetectGPUs. Si nil, les GPU de Base.Caps sont conservés.
	GPUProbe func(ctx context.Context) []capability.GPUInfo

	client *http.Client
	now    func() time.Time
}

// New crée un agent avec des valeurs par défaut raisonnables.
func New(serverURL string, base capability.Node) *Agent {
	return &Agent{
		ServerURL: strings.TrimRight(serverURL, "/"),
		Base:      base,
		Detector:  StaticDetector{State: capability.PresenceFree},
		Interval:  3 * time.Second,
		client:    &http.Client{Timeout: 5 * time.Second},
		now:       time.Now,
	}
}

// WithHTTPClient injecte un client (tests).
func (a *Agent) WithHTTPClient(c *http.Client) *Agent { a.client = c; return a }

// snapshot construit l'état courant du node à publier.
func (a *Agent) snapshot(ctx context.Context) capability.Node {
	n := a.Base.Clone()
	if a.Detector != nil {
		n.Presence = a.Detector.Detect()
	}
	if a.LoadedModels != nil {
		n.Caps.LoadedModels = a.LoadedModels(ctx)
	}
	if a.GPUProbe != nil {
		if gpus := a.GPUProbe(ctx); gpus != nil {
			n.Caps.GPUs = gpus
		}
	}
	n.LastHeartbeat = a.now()
	return n
}

// heartbeat publie un snapshot au serveur.
func (a *Agent) heartbeat(ctx context.Context) error {
	body, err := json.Marshal(a.snapshot(ctx))
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.ServerURL+"/internal/v1/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat: statut %d", resp.StatusCode)
	}
	return nil
}

// Run boucle jusqu'à annulation du contexte. Les erreurs de heartbeat sont
// renvoyées via onError sans interrompre la boucle (un raté réseau est normal).
func (a *Agent) Run(ctx context.Context, onError func(error)) error {
	ticker := time.NewTicker(a.Interval)
	defer ticker.Stop()

	// Premier heartbeat immédiat (enrôlement).
	if err := a.heartbeat(ctx); err != nil && onError != nil {
		onError(err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := a.heartbeat(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}
