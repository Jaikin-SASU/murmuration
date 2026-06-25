package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/Jaikin-SASU/murmuration/pkg/capability"
)

func TestParseNvidiaSMI(t *testing.T) {
	out := "NVIDIA GeForce RTX 4090, 24564, 20480, 5\nNVIDIA GeForce RTX 3060, 12288, 12000, 0\n"
	gpus := parseNvidiaSMI(out)
	if len(gpus) != 2 {
		t.Fatalf("attendu 2 GPU, obtenu %d", len(gpus))
	}
	if gpus[0].Vendor != capability.VendorNVIDIA || gpus[0].Model != "NVIDIA GeForce RTX 4090" {
		t.Fatalf("GPU 0 inattendu: %+v", gpus[0])
	}
	if gpus[0].VRAMTotalMB != 24564 || gpus[0].VRAMFreeMB != 20480 {
		t.Fatalf("VRAM GPU 0 incorrecte: %+v", gpus[0])
	}
	if gpus[0].UtilizationPct != 5 {
		t.Fatalf("utilisation GPU 0 = %v", gpus[0].UtilizationPct)
	}
}

func TestParseNvidiaSMIEmpty(t *testing.T) {
	if gpus := parseNvidiaSMI("\n  \n"); len(gpus) != 0 {
		t.Fatalf("attendu 0 GPU, obtenu %d", len(gpus))
	}
}

func TestParseAppleProfiler(t *testing.T) {
	out := []byte(`{"SPDisplaysDataType":[{"_name":"Apple M2 Pro","sppci_model":"Apple M2 Pro"}]}`)
	gpus := parseAppleProfiler(out, 32768)
	if len(gpus) != 1 {
		t.Fatalf("attendu 1 GPU, obtenu %d", len(gpus))
	}
	if gpus[0].Vendor != capability.VendorApple || gpus[0].Model != "Apple M2 Pro" {
		t.Fatalf("GPU Apple inattendu: %+v", gpus[0])
	}
	if gpus[0].VRAMTotalMB != 32768 {
		t.Fatalf("VRAM unifiée = %d, attendu 32768", gpus[0].VRAMTotalMB)
	}
}

func TestDetectGPUsPrefersNvidia(t *testing.T) {
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "nvidia-smi":
			return []byte("NVIDIA RTX A4000, 16376, 16000, 0\n"), nil
		default:
			return nil, errors.New("non appelé")
		}
	}
	gpus := DetectGPUs(context.Background(), run)
	if len(gpus) != 1 || gpus[0].Vendor != capability.VendorNVIDIA {
		t.Fatalf("attendu un GPU NVIDIA, obtenu %+v", gpus)
	}
}

func TestDetectGPUsFallsBackToApple(t *testing.T) {
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "nvidia-smi":
			return nil, errors.New("introuvable")
		case "system_profiler":
			return []byte(`{"SPDisplaysDataType":[{"sppci_model":"Apple M3 Max"}]}`), nil
		case "sysctl":
			return []byte("68719476736"), nil // 64 Gio
		}
		return nil, errors.New("inattendu")
	}
	gpus := DetectGPUs(context.Background(), run)
	if len(gpus) != 1 || gpus[0].Vendor != capability.VendorApple {
		t.Fatalf("attendu un GPU Apple, obtenu %+v", gpus)
	}
	if gpus[0].VRAMTotalMB != 65536 {
		t.Fatalf("VRAM unifiée = %d, attendu 65536", gpus[0].VRAMTotalMB)
	}
}

func TestDetectGPUsCPUOnly(t *testing.T) {
	run := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return nil, errors.New("aucune commande GPU")
	}
	if gpus := DetectGPUs(context.Background(), run); gpus != nil {
		t.Fatalf("attendu nil (CPU-only), obtenu %+v", gpus)
	}
}

func TestSnapshotUsesGPUProbe(t *testing.T) {
	a := New("http://unused", capability.Node{ID: "n1"})
	a.GPUProbe = func(context.Context) []capability.GPUInfo {
		return []capability.GPUInfo{{Vendor: capability.VendorNVIDIA, VRAMTotalMB: 24000, VRAMFreeMB: 18000}}
	}
	snap := a.snapshot(context.Background())
	if len(snap.Caps.GPUs) != 1 || snap.Caps.GPUs[0].VRAMFreeMB != 18000 {
		t.Fatalf("le snapshot devrait refléter le probe GPU: %+v", snap.Caps.GPUs)
	}
}
