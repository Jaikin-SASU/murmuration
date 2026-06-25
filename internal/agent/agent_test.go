package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jaikin-SASU/murmuration/pkg/capability"
)

func TestPresenceIdleBased(t *testing.T) {
	open := true
	idle := 1 * time.Minute
	d := IdleBased{
		ActiveThreshold: 5 * time.Minute,
		IdleInput:       func() time.Duration { return idle },
		SessionOpen:     func() bool { return open },
	}

	if got := d.Detect(); got != capability.PresenceActive {
		t.Fatalf("input récent → attendu Active, obtenu %s", got)
	}
	idle = 10 * time.Minute
	if got := d.Detect(); got != capability.PresenceIdle {
		t.Fatalf("inactif → attendu Idle, obtenu %s", got)
	}
	open = false
	if got := d.Detect(); got != capability.PresenceFree {
		t.Fatalf("session fermée → attendu Free, obtenu %s", got)
	}
}

func TestSnapshotReflectsPresenceAndModels(t *testing.T) {
	a := New("http://unused", capability.Node{ID: "n1", Hostname: "poste"})
	a.Detector = StaticDetector{State: capability.PresenceIdle}
	a.LoadedModels = func(context.Context) []string { return []string{"llama3"} }

	snap := a.snapshot(context.Background())
	if snap.Presence != capability.PresenceIdle {
		t.Errorf("présence = %s", snap.Presence)
	}
	if len(snap.Caps.LoadedModels) != 1 || snap.Caps.LoadedModels[0] != "llama3" {
		t.Errorf("modèles = %v", snap.Caps.LoadedModels)
	}
}

func TestHeartbeatPostsSnapshot(t *testing.T) {
	var received capability.Node
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/heartbeat" {
			t.Errorf("chemin %q", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &received)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	a := New(srv.URL, capability.Node{ID: "n1"})
	a.Detector = StaticDetector{State: capability.PresenceFree}
	if err := a.heartbeat(context.Background()); err != nil {
		t.Fatal(err)
	}
	if received.ID != "n1" {
		t.Fatalf("le serveur a reçu ID = %q", received.ID)
	}
	if received.Presence != capability.PresenceFree {
		t.Fatalf("présence reçue = %s", received.Presence)
	}
}

func TestRunStopsOnContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := New(srv.URL, capability.Node{ID: "n1"})
	a.Interval = 10 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := a.Run(ctx, nil); err != context.DeadlineExceeded {
		t.Fatalf("Run devrait s'arrêter sur le contexte, err = %v", err)
	}
}
