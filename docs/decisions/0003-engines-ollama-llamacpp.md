# ADR-0003 — Ollama (répliques) + llama.cpp RPC (pipeline nuit)

- Statut : Accepté
- Date : 2026-06-25

## Contexte

Parc hétérogène (CUDA + Apple Metal + Windows + CPU-only), voir
[P5](../PROBLEMS-AND-CONSTRAINTS.md#p5). Aucun moteur unique ne couvre tout
proprement avec éclatement.

## Décision

Moteur **mixte selon le besoin**, derrière une abstraction `Backend`
(voir [ADR-0006](0006-future-proof-pluggable-backends.md)) :

- **Ollama** sur chaque poste pour le **pool de répliques** (jour). Installation
  simple Windows + Mac, API REST propre, gestion chargement/déchargement, CUDA +
  Metal. C'est le chemin par défaut.
- **llama.cpp RPC** (`llama-server` + `rpc-server`) pour le **pipeline 2-4 nœuds**
  en mode nuit : seul moteur flexible mélangeant CUDA + Metal + Windows + Vulkan
  dans un même groupe. Batch uniquement.

## Conséquences

- Mise en route rapide via Ollama, sans renoncer à l'éclatement (llama.cpp).
- Deux backends à maintenir, isolés derrière l'interface `Backend`.
- llama.cpp RPC est marqué « PoC, insecure » → confiné au LAN, mode nuit, nœuds
  de confiance.

## Déclencheur de réévaluation

- Apparition d'un moteur unique cross-vendeur stable avec bon pipeline → fusionner.
- Ajout de backends spécialisés (vLLM sur un nœud GPU costaud, MLX sur Mac) :
  l'abstraction `Backend` les accueille sans toucher au scheduler.
