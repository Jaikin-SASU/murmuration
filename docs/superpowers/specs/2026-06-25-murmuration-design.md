# Spec — Murmuration

- Date : 2026-06-25
- Projet : **Murmuration** (binaires/CLI : `murmur`)
- Org GitHub : **JAIKIN-SASU** · contributeur : **victorwhale** · licence : à confirmer (MIT proposé)
- Docs liées : [DESIGN](../../DESIGN.md) · [PROBLEMS-AND-CONSTRAINTS](../../PROBLEMS-AND-CONSTRAINTS.md) · [ADR](../../decisions/README.md)

## 1. Problème & objectif

Chez Kappeler, ~30 postes de bureau (Windows + Mac, pilotés par Active Directory)
sont sous-exploités. On veut **agréger leur puissance CPU/GPU/RAM en un pool de
calcul LLM distribué**, exposé via un endpoint OpenAI-compatible, qui :

- privilégie les gros GPU,
- donne une **priorité absolue à l'utilisateur assis au poste**,
- s'appuie sur l'AD (inventaire, déploiement, présence, auth),
- **gagne en puissance automatiquement** quand les modèles deviennent plus denses.

## 2. Contraintes dures (non négociables)

Issues du [journal](../../PROBLEMS-AND-CONSTRAINTS.md) :

- **Pas** d'agrégation mémoire transparente entre machines (P1).
- **Pas** d'éclatement interactif d'un très gros modèle sur le parc : 0,85–6 tok/s,
  inutilisable en chat (P2).
- **Pas** de tensor parallel sur LAN ordinaire (P3).
- L'utilisateur au poste ne doit **jamais** subir de ralentissement perceptible (P4).
- Parc hétérogène : CUDA + Apple Metal + Windows + CPU-only (P5).

## 3. Objectifs / Non-objectifs

**Objectifs (v1)** : pool de répliques, scheduler présence-aware, endpoint
OpenAI-compatible, intégration AD (inventaire + présence + auth), file batch
durable, déploiement Windows (GPO/MSI) + Mac (pkg/launchd), mode nuit pipeline
2-4 nœuds (batch).

**Non-objectifs (v1)** : sharding interactif ; tensor parallel ; sharding optimisé
par IA (track de recherche, [ADR-0007](../../decisions/0007-ai-optimized-sharding-research-track.md)).

> Note : l'utilisateur a exprimé « tout d'un bloc ». Le mode nuit pipeline et le
> déploiement GPO sont inclus en v1 mais isolés derrière des interfaces ; le plan
> d'implémentation pourra les séquencer en dernier pour dé-risquer.

## 4. Architecture

Trois rôles : **clients** → **control plane** (serveur JAIKIN, Go) → **agents**
(1/poste, Go) pilotant un **Backend** (Ollama / llama.cpp RPC). Schéma complet
dans [DESIGN](../../DESIGN.md#architecture).

### Tiers de routage
- Tier 0 — serveur JAIKIN : gros modèles, always-on.
- Tier 1 — répliques GPU Kappeler : gros GPU d'abord.
- Tier 2 — pool CPU/Apple : overflow, embeddings, batch.
- Tier nuit — pipeline RPC 2-4 nœuds : modèle un cran plus gros, batch.

## 5. Composants & interfaces

| Composant | Responsabilité | Interface clé |
|---|---|---|
| `gateway` | Endpoint OpenAI-compatible, auth, idempotence, streaming | HTTP `/v1/*` |
| `scheduler` | Score capacité, routage tiers, auto-scaling | `Pick(req) → Node` |
| `registry` | État vivant des nodes & capabilities | `Upsert`, `Query` |
| `batch-queue` | File durable de tâches parallélisables | `Enqueue`, `Lease`, `Ack` |
| `ad-connector` | Inventaire, présence, identité (LDAP/Kerberos) | `FleetSource`, `IdentityProvider` |
| `agent` | Présence locale, télémétrie, pilote le Backend | gRPC stream |
| `backend` | Exécution d'inférence | `Backend` (Load/Unload/Generate/Embed) |
| `deploy` | Packaging GPO/MSI + pkg/launchd | artefacts CI |

**Interface `Backend`** (pluggable, cœur du 10x sans réécriture) :
`OllamaBackend`, `LlamaRpcBackend` en v1 ; `VllmBackend`/`MlxBackend`/futurs s'y
branchent sans toucher au scheduler ([ADR-0006](../../decisions/0006-future-proof-pluggable-backends.md)).

## 6. Scheduling & présence

Score : `poids_VRAM_libre × tier_GPU × dispo_présence − pénalité_réseau − charge`.
Trois niveaux de présence (Actif / Idle / Libre) avec garde-fou local sur
l'agent. Auto-scaling piloté par profondeur de file + latence. Détail complet et
table de politique dans [ADR-0005](../../decisions/0005-presence-aware-scheduling.md).

## 7. Intégration AD

Inventaire, déploiement, présence, auth (token adossé à l'AD). Le dispatch reste
au scheduler. Détail dans [ADR-0004](../../decisions/0004-active-directory-roles.md).
Interfaces `FleetSource` / `IdentityProvider` pour rester utilisable hors-AD (OSS).

## 8. Gestion d'erreurs

- Node injoignable → sortie du pool + réémission idempotente ; jamais d'échec
  silencieux.
- Échec de chargement modèle → node inéligible pour ce modèle, log détaillé.
- Auth échouée → 401 explicite, aucune fuite d'info.
- Validation stricte des entrées à la gateway (schémas typés).

## 9. Tests (cible ≥ 80 %)

- **Unitaires** : score scheduler, FSM présence, parsing télémétrie, normalisation API.
- **Intégration** : gateway↔scheduler↔registry avec agents simulés ; AD contre LDAP mocké.
- **E2E** : node réel (Ollama en conteneur) + requête `/v1/chat` bout-en-bout ;
  scénario « utilisateur revient → node retiré du pool ».

## 10. Les deux leviers de 10x

1. **Pari densité** (dès v1) — le pool récolte gratuitement les modèles qui
   rétrécissent ([ADR-0006](../../decisions/0006-future-proof-pluggable-backends.md)).
2. **Sharding optimisé par IA** (recherche) — un modèle (Fable/Mythos) planifie la
   découpe + spécule pour 10x les tok/s du mode nuit
   ([ADR-0007](../../decisions/0007-ai-optimized-sharding-research-track.md)).

## 11. Questions ouvertes

- Renommer le repo → `murmuration` sur JAIKIN-SASU.
- Auth : JWT adossé AD vs Kerberos direct.
- Persistance file batch : BoltDB/SQLite embarqué vs externe.
- Confirmer la licence (MIT proposé).
- Séquençage v1 « tout d'un bloc » vs cœur d'abord (à trancher dans le plan).
