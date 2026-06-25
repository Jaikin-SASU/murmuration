# Changelog

Toutes les versions notables de Murmuration. Format inspiré de
[Keep a Changelog](https://keepachangelog.com/fr/), versionnage [SemVer](https://semver.org/lang/fr/).

## [v0.2.0] — 2026-06-25

### Ajouté
- **Détection GPU automatique** sur l'agent : NVIDIA via `nvidia-smi`, Apple
  Silicon via `system_profiler` + mémoire unifiée. La VRAM libre est rafraîchie à
  chaque heartbeat, ce qui affine le score du scheduler en temps réel.
  Les flags `--gpu-vendor`/`--gpu-vram-mb` deviennent un override manuel.

### Feuille de route (prochaines versions)
- v0.3 — Connecteur Active Directory (inventaire + présence + auth).
- v0.3 — Mode nuit : pipeline llama.cpp RPC sur 2-4 nœuds.
- Détection de présence native Windows/macOS, file batch durable, packaging GPO/MSI.
- Recherche : sharding optimisé par IA (Fable/Mythos).

Voir les [issues](https://github.com/Jaikin-SASU/murmuration/issues) et
[milestones](https://github.com/Jaikin-SASU/murmuration/milestones).

## [v0.1.0] — 2026-06-25

### Ajouté
- Control plane (`murmur server`) : gateway OpenAI-compatible (chat + models,
  streaming SSE), scheduler présence-aware « gros GPU d'abord », registry à
  expiration par heartbeat, auth par token.
- Agent (`murmur agent`) : superviseur local cross-plateforme, garde-fou de
  présence, heartbeat + télémétrie, backend Ollama.
- Interface `backend.Backend` pluggable (Ollama + Fake).
- CI (vet + tests race + cross-build), Makefile, documentation (8 ADR).
