package registry

import (
	"testing"
	"time"

	"github.com/Jaikin-SASU/murmuration/pkg/capability"
)

func TestUpsertAndGet(t *testing.T) {
	r := New(10 * time.Second)
	r.Upsert(capability.Node{ID: "n1", Hostname: "poste-1"})

	n, ok := r.Get("n1")
	if !ok {
		t.Fatal("n1 devrait être présent")
	}
	if n.Hostname != "poste-1" {
		t.Fatalf("hostname = %q", n.Hostname)
	}
	if n.LastHeartbeat.IsZero() {
		t.Error("le heartbeat aurait dû être estampillé")
	}
}

func TestStaleNodeDisappears(t *testing.T) {
	r := New(5 * time.Second)
	current := time.Unix(1000, 0)
	r.SetClock(func() time.Time { return current })

	r.Upsert(capability.Node{ID: "n1"})

	// Avance au-delà du TTL.
	current = current.Add(6 * time.Second)

	if _, ok := r.Get("n1"); ok {
		t.Error("n1 aurait dû être considéré mort (heartbeat expiré)")
	}
	if got := len(r.List()); got != 0 {
		t.Errorf("List() = %d, attendu 0", got)
	}
}

func TestPrune(t *testing.T) {
	r := New(5 * time.Second)
	current := time.Unix(1000, 0)
	r.SetClock(func() time.Time { return current })

	r.Upsert(capability.Node{ID: "old"})
	current = current.Add(10 * time.Second)
	r.Upsert(capability.Node{ID: "fresh"})

	if removed := r.Prune(); removed != 1 {
		t.Fatalf("Prune a retiré %d nodes, attendu 1", removed)
	}
	if _, ok := r.Get("fresh"); !ok {
		t.Error("le node frais aurait dû survivre")
	}
}

func TestListReturnsClones(t *testing.T) {
	r := New(10 * time.Second)
	r.Upsert(capability.Node{
		ID:   "n1",
		Caps: capability.Capabilities{LoadedModels: []string{"llama3"}},
	})

	list := r.List()
	list[0].Caps.LoadedModels[0] = "muté"

	again := r.List()
	if again[0].Caps.LoadedModels[0] != "llama3" {
		t.Error("List() devrait renvoyer des clones isolés")
	}
}
