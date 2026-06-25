# Murmuration — Cluster de calcul LLM distribué (parc de bureau + AD)

> Document vivant. Projet : **Murmuration** (CLI/binaires : `murmur`).
> Org GitHub : **JAIKIN-SASU** · contributeur : **victorwhale**.
> *La nuée d'étourneaux : 30 machines faibles qui se meuvent comme un seul organisme.*
> Voir aussi : [PROBLEMS-AND-CONSTRAINTS](PROBLEMS-AND-CONSTRAINTS.md) ·
> [décisions (ADR)](decisions/README.md).

Dernière mise à jour : 2026-06-25

## Vision en une phrase

Transformer un parc de ~30 postes de bureau hétérogènes (Windows + Mac, pilotés
par AD) en un **pool de calcul LLM distribué**, exposé via un endpoint
OpenAI-compatible, qui privilégie les gros GPU, respecte l'utilisateur assis au
poste, et **gagne en puissance tout seul** à mesure que les modèles deviennent
plus denses ([ADR-0006](decisions/0006-future-proof-pluggable-backends.md)).

## Ce qu'on NE fait PAS (et pourquoi)

- Pas d'agrégation mémoire transparente entre machines ([P1](PROBLEMS-AND-CONSTRAINTS.md#p1)).
- Pas d'éclatement interactif d'un très gros modèle sur le parc ([P2](PROBLEMS-AND-CONSTRAINTS.md#p2)).
- Pas de tensor parallel sur LAN ordinaire ([P3](PROBLEMS-AND-CONSTRAINTS.md#p3)).

## Architecture

```
   Clients (apps, scripts, curl)
            │  HTTP  API OpenAI-compatible  + auth AD
            ▼
   ┌──────────────────────────────────────────────┐
   │  CONTROL PLANE (serveur JAIKIN, Go)           │
   │  • Gateway (contrat public stable)            │
   │  • Scheduler policy-driven, routé par tiers   │
   │  • Capability registry (état du parc)         │
   │  • File batch durable                         │
   │  • Connecteur AD (LDAP / Kerberos)            │
   └──────────────────────────────────────────────┘
        │ gRPC bidirectionnel (heartbeat, télémétrie, ordres)
        ▼
   ┌──────────┐  ┌──────────┐  ┌──────────┐
   │  AGENT   │  │  AGENT   │  │  AGENT   │   1 par poste (30×), Go
   │ presence │  │ presence │  │ presence │   garde-fou utilisateur local
   │ + Backend│  │ + Backend│  │ + Backend│
   │  ▼       │  │  ▼       │  │  ▼       │
   │ Ollama   │  │ Ollama   │  │ llama.cpp│
   │ (jour)   │  │ (jour)   │  │ RPC(nuit)│
   └──────────┘  └──────────┘  └──────────┘
```

### Tiers de routage

- **Tier 0** — serveur JAIKIN : gros modèles, latence faible, always-on.
- **Tier 1** — répliques GPU Kappeler : postes idle/libres, gros GPU d'abord.
- **Tier 2** — pool CPU/Apple : overflow, embeddings, batch.
- **Tier nuit** — groupe pipeline RPC 2-4 nœuds : modèle un cran plus gros, batch.

## Composants (unités isolées, testables séparément)

| Composant | Rôle | Dépend de |
|---|---|---|
| **gateway** | Endpoint OpenAI-compatible, auth, idempotence, streaming | scheduler |
| **scheduler** | Score capacité, routage par tiers, auto-scaling | registry, télémétrie |
| **registry** | État vivant des nodes & capabilities | heartbeats agents |
| **batch-queue** | File durable de tâches parallélisables | registry |
| **ad-connector** | Inventaire, présence, identité (LDAP/Kerberos) | AD Kappeler |
| **agent** | Superviseur local : présence, télémétrie, pilote Backend | Backend, OS |
| **backend (iface)** | `OllamaBackend`, `LlamaRpcBackend`, futurs… | moteurs |
| **deploy** | Packaging GPO/MSI (Win) + pkg/launchd (Mac) | — |

## Flux d'une requête (jour, pool de répliques)

1. Client → `POST /v1/chat/completions` avec token AD.
2. Gateway authentifie, normalise, demande un node au scheduler.
3. Scheduler choisit le meilleur node `Libre/Idle` ayant le modèle chargé
   (sinon scale-up : chauffe un node).
4. Gateway streame la réponse depuis le Backend du node ; si le node tombe en
   vol → réémission idempotente vers un autre.

## Gestion d'erreurs (principes)

- Node injoignable → sortie du pool + réémission ; jamais d'erreur silencieuse.
- Échec de chargement modèle → node marqué inéligible pour ce modèle, log détaillé.
- Auth échouée → 401 explicite, pas de fuite d'info.
- Validation stricte des entrées à la frontière (gateway), schémas typés.

## Stratégie de tests

- **Unitaires** : score scheduler, machine à états de présence, parsing
  télémétrie, normalisation API. Cible ≥ 80 %.
- **Intégration** : gateway↔scheduler↔registry avec agents simulés ; connecteur
  AD contre un LDAP mocké.
- **E2E** : un faux node (Ollama réel en conteneur) + une requête `/v1/chat`
  bout-en-bout ; scénario « utilisateur revient → node retiré ».

## Deux leviers de 10x

1. **Pari densité** ([ADR-0006](decisions/0006-future-proof-pluggable-backends.md))
   — le pool de répliques récolte gratuitement les modèles qui rétrécissent.
   Disponible dès v1, sans effort de recherche.
2. **Sharding optimisé par l'IA** ([ADR-0007](decisions/0007-ai-optimized-sharding-research-track.md))
   — piste de recherche : un modèle (Fable/Mythos) planifie la découpe, spécule
   et masque la latence pour 10x les tok/s d'un groupe nuit. Branché sur
   l'interface `Backend`, n'impacte pas le cœur v1.

## Questions ouvertes

- Renommer le repo `GPU-COMPANY-CLUSTER` → `murmuration` sur l'org JAIKIN-SASU.
- Format d'auth précis : token JWT adossé à l'AD vs Kerberos direct.
- Persistance de la file batch : embarquée (BoltDB/SQLite) vs externe.
- Périmètre exact v1 vs v2 (l'utilisateur a demandé « tout d'un bloc » — à
  arbitrer dans le plan d'implémentation).
