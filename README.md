# Murmuration

> *La nuée d'étourneaux : des dizaines de machines de bureau faibles qui se meuvent comme un seul organisme.*

**Murmuration** transforme un parc de machines de bureau hétérogènes (Windows + macOS),
piloté par Active Directory, en un **pool de calcul LLM distribué**, exposé via un
**endpoint OpenAI-compatible**. Il privilégie les gros GPU, **donne la priorité absolue
à l'utilisateur assis au poste**, et est conçu pour gagner en puissance automatiquement
à mesure que les modèles deviennent plus denses.

Projet open-source · Org [JAIKIN-SASU](https://github.com/Jaikin-SASU) · Licence MIT.

---

## Pourquoi

Acheter des GPU cloud coûte cher et expédie vos données ailleurs. Pendant ce temps, un
parc de PME dort la nuit et tourne au ralenti le jour. Murmuration **récolte cette
capacité existante** pour servir des modèles d'IA en local, sous votre contrôle.

L'intention : **démocratiser le calcul IA**, réutiliser le matériel déjà payé, garder
les données sur site.

## Ce que Murmuration fait (et ne fait pas)

✅ **Pool de répliques** : chaque poste capable sert une copie d'un modèle qui tient
chez lui (7B–14B quantifié, plus sur les gros GPU) ; la gateway répartit les requêtes.
✅ **Priorité utilisateur** : un poste dont l'utilisateur est actif sort du pool.
✅ **Gros GPU d'abord**, auto-scaling par profondeur de file.
✅ **Endpoint OpenAI-compatible** : vos clients existants marchent sans changement.
✅ **Hétérogène** : NVIDIA (CUDA), Apple Silicon (Metal), CPU — via Ollama.

❌ **Pas** d'agrégation mémoire transparente entre machines (physiquement impossible).
❌ **Pas** d'inférence interactive d'un très gros modèle éclaté sur le parc : sur LAN
ordinaire, ça plafonne à 0,85–6 tok/s (utilisable en batch nuit uniquement, 2-4 nœuds).
❌ **Pas** de tensor parallel sur réseau ordinaire (réservé InfiniBand/NVLink).

Le détail chiffré est dans [`docs/PROBLEMS-AND-CONSTRAINTS.md`](docs/PROBLEMS-AND-CONSTRAINTS.md).

## Architecture

```
Clients ──HTTP (API OpenAI) ─▶  CONTROL PLANE (murmur server)
                                  gateway · scheduler · registry · file batch
                                          │ HTTP/JSON (heartbeat + télémétrie)
                                          ▼
                         AGENTS (murmur agent, 1/poste) ─▶ Ollama / llama.cpp RPC
```

Détail dans [`docs/DESIGN.md`](docs/DESIGN.md) et les [ADR](docs/decisions/README.md).

## Démarrage rapide

Prérequis : [Go 1.26+](https://go.dev/dl/) et [Ollama](https://ollama.com) sur les postes.

```bash
# 1. Construire le binaire
make build      # → bin/murmur

# 2. Sur le serveur : lancer le control plane
bin/murmur server --addr :8080 --token MON_TOKEN

# 3. Sur chaque poste : lancer l'agent (Ollama doit tourner localement)
bin/murmur agent --server http://serveur:8080 \
  --ollama http://127.0.0.1:11434 \
  --gpu-vendor nvidia --gpu-vram-mb 24000 \
  --presence free
```

Appeler l'endpoint comme l'API OpenAI :

```bash
curl http://serveur:8080/v1/chat/completions \
  -H "Authorization: Bearer MON_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3:8b","messages":[{"role":"user","content":"Bonjour"}]}'
```

`GET /v1/models` liste les modèles chargés sur le parc. `stream:true` est supporté (SSE).

## Deux leviers de 10x

1. **Pari densité** (dès maintenant) — quand un 14B atteindra la qualité d'un 70B
   d'aujourd'hui, le même parc servira « du gros » sans rien changer.
   [ADR-0006](docs/decisions/0006-future-proof-pluggable-backends.md)
2. **Sharding optimisé par IA** (recherche) — un modèle planifie la découpe et masque
   la latence pour 10x les tok/s du mode nuit.
   [ADR-0007](docs/decisions/0007-ai-optimized-sharding-research-track.md)

## Développement

```bash
make test       # tests + couverture
make vet        # go vet
make cover      # rapport HTML de couverture
make dist       # binaires Windows / macOS / Linux
```

## Statut

**v0.2** — fondation + **détection GPU automatique** (NVIDIA / Apple Silicon).
v0.1 livrait le control plane, le scheduler présence-aware, la gateway OpenAI, le
backend Ollama et l'agent cross-plateforme. Détail dans le [CHANGELOG](CHANGELOG.md).
Prochaines étapes (v0.3) : connecteur Active Directory, mode nuit pipeline. Voir la
[feuille de route](docs/decisions/README.md) et les [issues](https://github.com/Jaikin-SASU/murmuration/issues).

## Contribuer

Voir [CONTRIBUTING.md](CONTRIBUTING.md). Les PR sont bienvenues.

## Licence

[MIT](LICENSE) © 2026 JAIKIN SASU.
