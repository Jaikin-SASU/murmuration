# ADR-0002 — Go pour le serveur et l'agent

- Statut : Accepté
- Date : 2026-06-25

## Contexte

L'agent doit tourner comme **service Windows** et **daemon macOS (launchd)** sur
~30 postes hétérogènes, démarrer son moteur local, sonder GPU/présence, et parler
au control plane. Le serveur central héberge gateway + scheduler + registry.

## Décision

**Go** pour les deux binaires (agent et serveur), mêmes conventions, build
cross-compilé par OS.

## Conséquences

- Binaire statique unique par OS → déploiement trivial (idéal GPO/MSI et pkg Mac).
- Excellente concurrence (heartbeats, streaming, fan-out scheduler).
- Pas d'écosystème Python IA côté serveur, mais on n'en a pas besoin :
  l'inférence est déléguée aux moteurs (Ollama/llama.cpp). Le serveur n'orchestre
  que du réseau et de la décision.

## Déclencheur de réévaluation

- Besoin d'embarquer de la logique ML lourde *dans* le control plane (peu
  probable, l'inférence reste dans les moteurs).
