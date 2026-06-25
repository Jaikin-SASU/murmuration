// Package capability décrit, de façon abstraite, ce qu'un node du parc sait
// faire. Le scheduler raisonne sur ces capacités, jamais sur un produit précis
// (Ollama, llama.cpp…). Voir ADR-0006.
package capability

import "time"

// GPUVendor identifie le fabricant d'un GPU.
type GPUVendor string

const (
	VendorNVIDIA GPUVendor = "nvidia"
	VendorAMD    GPUVendor = "amd"
	VendorApple  GPUVendor = "apple"
	VendorNone   GPUVendor = "none"
)

// OS identifie le système d'exploitation du poste.
type OS string

const (
	OSWindows OS = "windows"
	OSMacOS   OS = "darwin"
	OSLinux   OS = "linux"
)

// PresenceState : priorité absolue à l'utilisateur assis au poste (ADR-0005).
type PresenceState string

const (
	// PresenceActive : input récent / plein écran → node retiré du pool.
	PresenceActive PresenceState = "active"
	// PresenceIdle : session ouverte sans input → disponible mais plafonné.
	PresenceIdle PresenceState = "idle"
	// PresenceFree : session fermée / nuit → pleine puissance.
	PresenceFree PresenceState = "free"
)

// GPUInfo décrit un GPU et son état mémoire instantané.
type GPUInfo struct {
	Vendor         GPUVendor `json:"vendor"`
	Model          string    `json:"model"`
	VRAMTotalMB    int64     `json:"vram_total_mb"`
	VRAMFreeMB     int64     `json:"vram_free_mb"`
	UtilizationPct float64   `json:"utilization_pct"`
}

// Capabilities est la description qu'un node publie au control plane.
type Capabilities struct {
	OS           OS                `json:"os"`
	GPUs         []GPUInfo         `json:"gpus"`
	CPUCores     int               `json:"cpu_cores"`
	RAMTotalMB   int64             `json:"ram_total_mb"`
	RAMFreeMB    int64             `json:"ram_free_mb"`
	Backends     []string          `json:"backends"`      // ex. ["ollama","llama-rpc"]
	LoadedModels []string          `json:"loaded_models"` // modèles déjà en mémoire
	Endpoints    map[string]string `json:"endpoints"`     // backend -> base URL joignable
}

// Node est un membre du parc avec sa télémétrie vivante.
type Node struct {
	ID               string        `json:"id"`
	Hostname         string        `json:"hostname"`
	Caps             Capabilities  `json:"capabilities"`
	Presence         PresenceState `json:"presence"`
	InflightRequests int           `json:"inflight_requests"`
	QueueDepth       int           `json:"queue_depth"`
	LastHeartbeat    time.Time     `json:"last_heartbeat"`
}

// BestGPU renvoie le GPU disposant de la plus grande VRAM totale.
func (c Capabilities) BestGPU() (GPUInfo, bool) {
	var best GPUInfo
	found := false
	for _, g := range c.GPUs {
		if !found || g.VRAMTotalMB > best.VRAMTotalMB {
			best = g
			found = true
		}
	}
	return best, found
}

// FreeVRAMMB somme la VRAM libre de tous les GPU.
func (c Capabilities) FreeVRAMMB() int64 {
	var total int64
	for _, g := range c.GPUs {
		total += g.VRAMFreeMB
	}
	return total
}

// HasModel indique si un modèle est déjà chargé sur le node.
func (c Capabilities) HasModel(model string) bool {
	for _, m := range c.LoadedModels {
		if m == model {
			return true
		}
	}
	return false
}

// SupportsBackend indique si le node expose un backend donné.
func (c Capabilities) SupportsBackend(name string) bool {
	for _, b := range c.Backends {
		if b == name {
			return true
		}
	}
	return false
}

// Clone renvoie une copie profonde (immuabilité : ADR-style maison, voir
// ecc-common-style). Les slices/maps sont recopiées pour éviter tout partage.
func (c Capabilities) Clone() Capabilities {
	out := c
	out.GPUs = append([]GPUInfo(nil), c.GPUs...)
	out.Backends = append([]string(nil), c.Backends...)
	out.LoadedModels = append([]string(nil), c.LoadedModels...)
	if c.Endpoints != nil {
		out.Endpoints = make(map[string]string, len(c.Endpoints))
		for k, v := range c.Endpoints {
			out.Endpoints[k] = v
		}
	}
	return out
}

// Clone renvoie une copie profonde du node.
func (n Node) Clone() Node {
	out := n
	out.Caps = n.Caps.Clone()
	return out
}
