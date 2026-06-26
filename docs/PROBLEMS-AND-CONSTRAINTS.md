# Journal des problèmes & contraintes

> Journal vivant. On consigne ici chaque mur technique rencontré, avec des
> chiffres mesurés quand on en a. Objectif : quand des modèles/matériels plus
> performants arriveront, on relit ce journal et on sait *exactement* quelles
> hypothèses réévaluer pour faire un 10x.

Dernière mise à jour : 2026-06-26

---

## Contexte

- Parc : ~30 postes PC (Windows) + Mac, machines de bureau **faibles**.
- Réseau : LAN ordinaire, **globalement lent** (hypothèse 1 GbE ou moins, parfois Wi-Fi).
- Pilotés par un serveur Windows **Active Directory** (Kappeler).
- Serveur central séparé (**JAIKIN**) qui héberge les gros modèles et le control plane.
- But initial exprimé : « agréger la puissance CPU/GPU/RAM des postes pour faire
  tourner de l'inférence LLM via un endpoint ».

---

## P1 — On ne peut PAS agréger la RAM/le GPU de plusieurs machines de façon transparente

**Problème.** L'idée intuitive « le gros modèle vit sur le serveur, les postes
prêtent juste leur RAM/GPU » n'est pas réalisable. Le calcul doit se faire là où
se trouvent les poids. « Emprunter » de la mémoire distante pour de l'inférence
n'existe pas : il faut envoyer *une part du modèle* sur le poste et l'y exécuter.

**Conséquence.** Toute approche « mémoire partagée réseau » est écartée.
→ voir [ADR-0001](decisions/0001-replica-pool-over-sharding.md).

---

## P2 — Éclater UN très gros modèle sur ~30 postes faibles = inutilisable en interactif

**Mesures (clusters grand public, 2025-2026).**

- Modèle classe 70B éclaté sur machines faibles : **0,85 à 6 tokens/s**.
  Utilisable en **batch asynchrone**, pas en chat fluide.
- Le goulot **n'est PAS la bande passante** : en pipeline, seul le vecteur
  d'activation traverse le réseau, soit ~**16 Ko/token** (FP16, hidden=8192),
  ~**8 Ko/token** si activations quantifiées 8-bit (Petals). Un seul lien 1 GbE
  (125 Mo/s) porterait des milliers de tokens/s de pur trafic d'activations.
- Le vrai goulot = **compute par machine faible** + **latence en série**.
  Décodage autorégressif = chaque token traverse les N machines l'une après
  l'autre. Plafonds réseau seuls (compute ignorée) :
  - filaire ~0,3 ms/saut, 30 sauts → ~110 tok/s
  - filaire ~1 ms/saut → ~33 tok/s
  - Wi-Fi ~5 ms/saut → ~6-7 tok/s
  La compute des postes faibles fait ensuite chuter le réel bien en dessous.

**Sources.**
- Petals, physique réseau : https://ar5iv.labs.arxiv.org/html/2209.01188
- prima.cpp (ring + prefetch, ~1,5 tok/s 70B) : https://arxiv.org/abs/2504.08791
- llama.cpp RPC (6,1 tok/s en 2.5GbE) : https://github.com/ggml-org/llama.cpp/discussions/9136
- Benchmark cluster Pi (0,85 tok/s) : https://www.jeffgeerling.com/blog/2025/i-regret-building-3000-pi-ai-cluster

---

## P3 — Le tensor parallel est exclu sur LAN ordinaire

**Problème.** Le tensor parallel exige **2 all-reduce synchrones bloquants par
couche et par token** (~160 round-trips pour 80 couches). Sur le chemin
critique → nécessite NVLink/InfiniBand (ou ≥10 GbE).

**Guidance vLLM** : « Efficient tensor parallelism requires fast internode
communication, preferably through high-speed network adapters such as
InfiniBand. »

**Conséquence.** Pour tout éclatement de modèle, on n'utilise **que du pipeline /
ring parallel**, et seulement sur **2 à 4 nœuds stables** (au-delà, la latence
série tue le gain). → voir [ADR-0003](decisions/0003-engines-ollama-llamacpp.md).

---

## P4 — Priorité absolue à l'utilisateur assis au poste

**Problème.** Un poste est avant tout l'outil de travail de quelqu'un. Lui voler
le GPU/CPU pendant qu'il travaille est inacceptable.

**Conséquence.** Garde-fou de présence local sur chaque agent + politique à 3
niveaux (actif / idle / libre). → voir
[ADR-0005](decisions/0005-presence-aware-scheduling.md).

---

## P5 — Parc hétérogène (NVIDIA + Apple Silicon + Windows + CPU-only)

**Problème.** Aucun moteur unique ne couvre proprement CUDA + Metal + Windows +
CPU avec éclatement. vLLM = pas de Windows/Mac. exo = Mac-centré. Ollama = pas
d'éclatement cross-machine.

**Conséquence.** Couche d'abstraction `Backend` + Ollama (répliques, partout) et
llama.cpp RPC (pipeline nuit, seul flexible multi-vendeurs).
→ voir [ADR-0003](decisions/0003-engines-ollama-llamacpp.md).

---

## P6 — CPU-only ne veut pas dire gros modèle dense classique

**Problème.** Les postes CPU-only restent trop lents pour servir confortablement
des modèles denses classiques en chat interactif. Les gains annoncés par
BitNet/bitnet.cpp concernent des modèles 1.58-bit avec kernels spécialisés, pas
une conversion gratuite de n'importe quel modèle existant.

**Conséquence.** On garde les CPU-only dans le Tier 2, orientés overflow, batch,
embeddings/classification et modèles 1.58-bit explicitement compatibles.
→ voir [ADR-0009](decisions/0009-cpu-native-1bit-bitnet-backend.md).

---

## Hypothèses à réévaluer quand le matériel/les modèles progressent

| Hypothèse | Si elle change… | Impact |
|---|---|---|
| LAN ≤ 1 GbE | Passage 10/25 GbE ou Thunderbolt mesh | Tensor parallel redevient envisageable (P3) |
| Postes faibles | GPU dédiés généralisés | Plus de répliques de gros modèles en local |
| Capacité/Go des modèles | Modèles 14B = qualité 200B d'aujourd'hui | **Le 10x principal** : le pool fait tourner du « gros » sans rien changer |
| Modèles 1.58-bit CPU-native | Qualité suffisante sur 7B-100B BitNet | Les postes CPU-only deviennent utiles sans GPU pour batch/overflow |
