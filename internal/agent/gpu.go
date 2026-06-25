package agent

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Jaikin-SASU/murmuration/pkg/capability"
)

// CommandRunner exécute une commande externe. Injectable pour les tests.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// DetectGPUs sonde le matériel : NVIDIA d'abord (nvidia-smi), puis Apple
// (system_profiler). Renvoie nil si rien n'est détecté (node CPU-only).
func DetectGPUs(ctx context.Context, run CommandRunner) []capability.GPUInfo {
	if run == nil {
		run = execRunner
	}
	if gpus := detectNvidia(ctx, run); len(gpus) > 0 {
		return gpus
	}
	if gpus := detectApple(ctx, run); len(gpus) > 0 {
		return gpus
	}
	return nil
}

func detectNvidia(ctx context.Context, run CommandRunner) []capability.GPUInfo {
	out, err := run(ctx, "nvidia-smi",
		"--query-gpu=name,memory.total,memory.free,utilization.gpu",
		"--format=csv,noheader,nounits")
	if err != nil {
		return nil
	}
	return parseNvidiaSMI(string(out))
}

// parseNvidiaSMI lit la sortie CSV de nvidia-smi (une ligne par GPU) :
// "NVIDIA GeForce RTX 4090, 24564, 20480, 5"
func parseNvidiaSMI(out string) []capability.GPUInfo {
	var gpus []capability.GPUInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			continue
		}
		g := capability.GPUInfo{
			Vendor:      capability.VendorNVIDIA,
			Model:       strings.TrimSpace(fields[0]),
			VRAMTotalMB: atoiSafe(fields[1]),
			VRAMFreeMB:  atoiSafe(fields[2]),
		}
		if len(fields) >= 4 {
			g.UtilizationPct = atofSafe(fields[3])
		}
		gpus = append(gpus, g)
	}
	return gpus
}

func detectApple(ctx context.Context, run CommandRunner) []capability.GPUInfo {
	out, err := run(ctx, "system_profiler", "SPDisplaysDataType", "-json")
	if err != nil {
		return nil
	}
	// La VRAM d'un Apple Silicon = mémoire unifiée (sysctl hw.memsize).
	var memMB int64
	if mem, err := run(ctx, "sysctl", "-n", "hw.memsize"); err == nil {
		memMB = atoiSafe(string(mem)) / (1024 * 1024)
	}
	return parseAppleProfiler(out, memMB)
}

// parseAppleProfiler lit le JSON de system_profiler. La mémoire unifiée est
// passée séparément (heuristique : Metal peut adresser l'essentiel de la RAM).
func parseAppleProfiler(out []byte, unifiedMB int64) []capability.GPUInfo {
	var payload struct {
		Displays []struct {
			Name      string `json:"_name"`
			Model     string `json:"sppci_model"`
			VendorRaw string `json:"spdisplays_vendor"`
		} `json:"SPDisplaysDataType"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil
	}
	var gpus []capability.GPUInfo
	for _, d := range payload.Displays {
		model := d.Model
		if model == "" {
			model = d.Name
		}
		if model == "" {
			continue
		}
		gpus = append(gpus, capability.GPUInfo{
			Vendor:      capability.VendorApple,
			Model:       model,
			VRAMTotalMB: unifiedMB,
			VRAMFreeMB:  unifiedMB,
		})
	}
	return gpus
}

func atoiSafe(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func atofSafe(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
