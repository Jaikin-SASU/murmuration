# ADR-0006 — Architecture pluggable + pari sur la densité des modèles

- Statut : Proposé
- Date : 2026-06-25

## Contexte

L'utilisateur veut une fondation qui pourra **faire un 10x quand des modèles plus
intelligents arriveront**, sans réécriture. La tentation naïve (éclater le gros
modèle du jour sur le parc) se heurte au mur de la physique
([P2](../PROBLEMS-AND-CONSTRAINTS.md#p2), [P3](../PROBLEMS-AND-CONSTRAINTS.md#p3)).
Il faut donc une invention *structurelle*, pas un hack de plus.

## Le pari (la thèse)

> **La capacité par gigaoctet des modèles augmente plus vite que tout le reste.**
> Un modèle de 14B aujourd'hui rivalise avec un 70B d'il y a 18 mois. Cette
> tendance est régulière et rapide.

Conséquence stratégique : **on ne parie pas sur le réseau, on parie sur les
modèles qui rétrécissent.** Le même pool de répliques qui sert un 14B aujourd'hui
servira, dans 12-18 mois, un modèle aussi capable que les 70-200B d'aujourd'hui —
**parce qu'il tiendra dans la même VRAM**. L'architecture ne change pas ; seuls
les fichiers de modèles changent. Le 10x arrive *tout seul* si la fondation est
model-agnostic.

## Décision

Construire un control plane **model-agnostic, contrat stable, backends
pluggables**, en 4 couches étanches :

1. **Gateway — contrat public stable** : API OpenAI-compatible
   (`/v1/chat/completions`, `/v1/embeddings`, `/v1/models`). Ne change jamais
   quand l'interne évolue. Les clients d'aujourd'hui marchent avec le cluster de
   demain.

2. **Capability registry — description abstraite** : chaque node publie ses
   *capabilities* `{VRAM, tier GPU, OS, modèles chargés, état présence, backends
   supportés}`. Aucune mention d'« Ollama » : on raisonne en capacités, pas en
   produits.

3. **Scheduler policy-driven, routé par tiers** :
   - *Tier 0* — serveur JAIKIN (gros modèles, latence faible, always-on)
   - *Tier 1* — répliques GPU Kappeler (postes idle/libres, gros GPU d'abord)
   - *Tier 2* — pool CPU/Apple Kappeler (overflow, embeddings, batch,
     backends CPU-native type BitNet/bitnet.cpp)
   - *Tier nuit* — groupe pipeline RPC 2-4 nœuds (un cran plus gros, batch)
   La politique est une donnée, pas du code en dur.

4. **Backends pluggables** derrière une interface `Backend` unique :
   `OllamaBackend` et `LlamaRpcBackend` au départ ; `BitnetCppBackend`,
   `VllmBackend`, `MlxBackend`, `XBackend` futurs s'y branchent **sans toucher au scheduler**.

Plus une **file batch durable** : les tâches parallélisables (embeddings, indexing
RAG, classification, génération par lots) sont drainées opportunistiquement par le
parc — le cas d'usage où 30 machines faibles brillent vraiment.

## Pourquoi ça permet le 10x

- **Contrat stable** → aucun client à migrer quand l'interne change.
- **Backends pluggables** → on adopte le meilleur moteur du moment sans réécrire.
- **Routage par capacités** → quand les nodes deviennent plus puissants OU les
  modèles plus denses, davantage de charge bascule automatiquement vers le pool.
- **Pari densité** → le travail le plus dur (rendre les modèles meilleurs) est
  fait par l'industrie ; notre fondation ne fait que *récolter* le gain.
- **CPU-native 1.58-bit** ([ADR-0009](0009-cpu-native-1bit-bitnet-backend.md))
  → si BitNet tient ses promesses sur des modèles utiles, les postes sans GPU
  rejoignent le pool pour batch/overflow sans changement de contrat public.

## Conséquences

- Discipline d'architecture forte (interfaces étanches) dès le départ.
- Un peu plus d'abstraction que le strict minimum d'un MVP — assumé : c'est le
  prix du 10x sans réécriture.

## Déclencheur de réévaluation

- Si la thèse de densité s'essouffle (modèles qui stagnent en qualité/Go),
  rebasculer l'effort vers l'interconnexion réseau (10/25 GbE) et l'éclatement.
