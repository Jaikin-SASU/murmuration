package scheduler

import (
	"testing"

	"github.com/Jaikin-SASU/murmuration/pkg/capability"
)

func node(id string, presence capability.PresenceState, vendor capability.GPUVendor, vramTotal, vramFree int64, models ...string) capability.Node {
	return capability.Node{
		ID:       id,
		Presence: presence,
		Caps: capability.Capabilities{
			GPUs:         []capability.GPUInfo{{Vendor: vendor, VRAMTotalMB: vramTotal, VRAMFreeMB: vramFree}},
			Backends:     []string{"ollama"},
			LoadedModels: models,
		},
	}
}

func TestActiveNodeExcluded(t *testing.T) {
	s := New(DefaultPolicy())
	n := node("busy", capability.PresenceActive, capability.VendorNVIDIA, 24000, 24000, "llama3")
	if s.Eligible(n, Request{Model: "llama3"}) {
		t.Fatal("un node Actif ne doit jamais être éligible (priorité utilisateur)")
	}
	if _, err := s.Pick([]capability.Node{n}, Request{Model: "llama3"}); err != ErrNoCapacity {
		t.Fatalf("attendu ErrNoCapacity, obtenu %v", err)
	}
}

func TestBigGPUFirst(t *testing.T) {
	s := New(DefaultPolicy())
	small := node("small", capability.PresenceFree, capability.VendorNVIDIA, 8000, 8000, "llama3")
	big := node("big", capability.PresenceFree, capability.VendorNVIDIA, 24000, 24000, "llama3")

	d, err := s.Pick([]capability.Node{small, big}, Request{Model: "llama3"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Node.ID != "big" {
		t.Fatalf("attendu le gros GPU 'big', obtenu %q", d.Node.ID)
	}
}

func TestPrefersLoadedModel(t *testing.T) {
	s := New(DefaultPolicy())
	// 'cold' a un plus gros GPU mais pas le modèle ; 'warm' l'a déjà chargé.
	cold := node("cold", capability.PresenceFree, capability.VendorNVIDIA, 24000, 24000)
	warm := node("warm", capability.PresenceFree, capability.VendorNVIDIA, 8000, 8000, "llama3")

	d, err := s.Pick([]capability.Node{cold, warm}, Request{Model: "llama3"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Node.ID != "warm" {
		t.Fatalf("attendu 'warm' (modèle déjà chargé), obtenu %q", d.Node.ID)
	}
	if d.NeedsLoad {
		t.Error("NeedsLoad devrait être false pour un node qui a le modèle")
	}
}

func TestScaleUpWhenNoneLoaded(t *testing.T) {
	s := New(DefaultPolicy())
	cold := node("cold", capability.PresenceFree, capability.VendorNVIDIA, 24000, 24000)

	d, err := s.Pick([]capability.Node{cold}, Request{Model: "mistral"})
	if err != nil {
		t.Fatal(err)
	}
	if !d.NeedsLoad {
		t.Error("attendu NeedsLoad=true (le modèle doit être chauffé)")
	}
}

func TestIdleIsPenalizedVsFree(t *testing.T) {
	s := New(DefaultPolicy())
	free := node("free", capability.PresenceFree, capability.VendorNVIDIA, 12000, 12000, "llama3")
	idle := node("idle", capability.PresenceIdle, capability.VendorNVIDIA, 12000, 12000, "llama3")

	if s.Score(free, Request{Model: "llama3"}) <= s.Score(idle, Request{Model: "llama3"}) {
		t.Error("un node Free doit scorer plus haut qu'un node Idle équivalent")
	}
}

func TestLoadPenaltyShiftsChoice(t *testing.T) {
	s := New(DefaultPolicy())
	loaded := node("loaded", capability.PresenceFree, capability.VendorNVIDIA, 12000, 12000, "llama3")
	loaded.InflightRequests = 20 // saturé
	idleFree := node("calm", capability.PresenceFree, capability.VendorNVIDIA, 12000, 12000, "llama3")

	d, err := s.Pick([]capability.Node{loaded, idleFree}, Request{Model: "llama3"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Node.ID != "calm" {
		t.Fatalf("la charge devrait faire préférer 'calm', obtenu %q", d.Node.ID)
	}
}

func TestUnsupportedBackendExcluded(t *testing.T) {
	s := New(DefaultPolicy())
	n := node("n", capability.PresenceFree, capability.VendorNVIDIA, 12000, 12000, "llama3")
	if _, err := s.Pick([]capability.Node{n}, Request{Model: "llama3", Backend: "vllm"}); err != ErrNoCapacity {
		t.Fatalf("attendu ErrNoCapacity pour backend non supporté, obtenu %v", err)
	}
}
