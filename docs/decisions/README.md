# Journal des décisions (ADR)

> Chaque décision technique structurante est consignée comme un ADR
> (Architecture Decision Record) : le **contexte**, la **décision**, les
> **conséquences**, et le **déclencheur de réévaluation** (qu'est-ce qui nous
> ferait changer d'avis). Format pensé pour le 10x : quand un modèle plus
> intelligent ou un meilleur matériel arrive, on relit les déclencheurs.

| # | Décision | Statut |
|---|---|---|
| [0001](0001-replica-pool-over-sharding.md) | Pool de répliques comme substrat, pas l'agrégation mémoire | Accepté |
| [0002](0002-go-everywhere.md) | Go pour le serveur et l'agent | Accepté |
| [0003](0003-engines-ollama-llamacpp.md) | Ollama (répliques) + llama.cpp RPC (pipeline nuit) | Accepté |
| [0004](0004-active-directory-roles.md) | Rôles de l'Active Directory | Accepté |
| [0005](0005-presence-aware-scheduling.md) | Scheduling conscient de la présence | Accepté |
| [0006](0006-future-proof-pluggable-backends.md) | Architecture pluggable + pari sur la densité des modèles | Proposé |
| [0007](0007-ai-optimized-sharding-research-track.md) | Sharding optimisé par l'IA (Fable/Mythos) | Proposé (recherche) |
| [0008](0008-http-json-transport-v1.md) | Transport HTTP/JSON pour la v1 (gRPC différé) | Accepté |
| [0009](0009-cpu-native-1bit-bitnet-backend.md) | Backend CPU-native 1.58-bit (BitNet/bitnet.cpp) | Proposé |

## Modèle d'ADR

```
# ADR-XXXX — Titre

- Statut : Proposé | Accepté | Remplacé par ADR-YYYY
- Date : YYYY-MM-DD

## Contexte
## Décision
## Conséquences
## Déclencheur de réévaluation
```
