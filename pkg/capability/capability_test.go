package capability

import "testing"

func sampleCaps() Capabilities {
	return Capabilities{
		OS: OSWindows,
		GPUs: []GPUInfo{
			{Vendor: VendorNVIDIA, Model: "RTX 3060", VRAMTotalMB: 12000, VRAMFreeMB: 8000},
			{Vendor: VendorNVIDIA, Model: "RTX 4090", VRAMTotalMB: 24000, VRAMFreeMB: 20000},
		},
		Backends:     []string{"ollama"},
		LoadedModels: []string{"llama3:8b"},
		Endpoints:    map[string]string{"ollama": "http://10.0.0.5:11434"},
	}
}

func TestBestGPU(t *testing.T) {
	g, ok := sampleCaps().BestGPU()
	if !ok {
		t.Fatal("attendu un GPU")
	}
	if g.Model != "RTX 4090" {
		t.Fatalf("BestGPU = %q, attendu RTX 4090", g.Model)
	}
}

func TestBestGPUEmpty(t *testing.T) {
	if _, ok := (Capabilities{}).BestGPU(); ok {
		t.Fatal("attendu aucun GPU")
	}
}

func TestFreeVRAM(t *testing.T) {
	if got := sampleCaps().FreeVRAMMB(); got != 28000 {
		t.Fatalf("FreeVRAMMB = %d, attendu 28000", got)
	}
}

func TestHasModelAndBackend(t *testing.T) {
	c := sampleCaps()
	if !c.HasModel("llama3:8b") {
		t.Error("llama3:8b devrait être chargé")
	}
	if c.HasModel("mistral") {
		t.Error("mistral ne devrait pas être chargé")
	}
	if !c.SupportsBackend("ollama") {
		t.Error("ollama devrait être supporté")
	}
	if c.SupportsBackend("llama-rpc") {
		t.Error("llama-rpc ne devrait pas être supporté")
	}
}

func TestCloneIsolation(t *testing.T) {
	orig := sampleCaps()
	clone := orig.Clone()
	clone.GPUs[0].VRAMFreeMB = 0
	clone.LoadedModels[0] = "changed"
	clone.Endpoints["ollama"] = "changed"

	if orig.GPUs[0].VRAMFreeMB != 8000 {
		t.Error("mutation du clone a fui sur l'original (GPUs)")
	}
	if orig.LoadedModels[0] != "llama3:8b" {
		t.Error("mutation du clone a fui sur l'original (LoadedModels)")
	}
	if orig.Endpoints["ollama"] != "http://10.0.0.5:11434" {
		t.Error("mutation du clone a fui sur l'original (Endpoints)")
	}
}
