# ADR-0009 — Backend CPU-native 1.58-bit (BitNet/bitnet.cpp)

- Statut : Proposé
- Date : 2026-06-26

## Contexte

Le pari densité d'[ADR-0006](0006-future-proof-pluggable-backends.md) ne passe
pas uniquement par la VRAM. Les modèles entraînés nativement en très basse
précision, notamment BitNet b1.58, déplacent une partie du gain vers les CPU :
moins de mémoire, moins d'énergie, kernels spécialisés, et exécution locale sur
des postes sans GPU.

Microsoft publie `bitnet.cpp`, framework d'inférence open-source pour modèles
1.58-bit. La documentation officielle annonce des accélérations CPU jusqu'à
6,17x sur x86, des réductions d'énergie jusqu'à 82,2 %, et la possibilité de
faire tourner un modèle BitNet b1.58 de 100B sur un seul CPU à vitesse de lecture
humaine. Ces chiffres sont à lire comme une opportunité d'architecture, pas
comme une garantie produit pour n'importe quel modèle : le gain dépend de modèles
entraînés/convertis pour BitNet, pas d'une quantification magique d'un modèle
dense existant.

Sources :
- bitnet.cpp officiel : https://github.com/microsoft/BitNet
- Rapport CPU 2024 : https://arxiv.org/abs/2410.16144
- Rapport bitnet.cpp 2025 : https://arxiv.org/abs/2502.11880

## Décision

Traiter `bitnet.cpp` comme un **backend CPU-native pluggable** derrière
l'interface `Backend`, au même titre qu'Ollama, llama.cpp RPC, vLLM ou MLX.

Positionnement cible :

- **Tier 2 CPU/Apple** : overflow, embeddings, classification, batch et tâches
  tolérant une latence plus haute.
- **Postes CPU-only** : regain d'intérêt si des modèles 1.58-bit utiles tiennent
  dans la RAM locale.
- **Batch nuit** : gros modèles BitNet sur CPU quand le coût énergétique et la
  disponibilité priment sur la latence interactive.

Nom de backend réservé dans les capabilities : `bitnet-cpp`.

## Conséquences

- Le control plane reste model-agnostic : un node annonce simplement
  `Backends:["bitnet-cpp"]`, ses modèles chargés, sa RAM et son endpoint.
- Le scheduler peut déjà filtrer par backend ; l'ajout réel consiste surtout à
  implémenter un adaptateur HTTP/CLI `BitnetCppBackend` et sa sonde agent.
- Les CPU-only ne deviennent pas équivalents aux gros GPU : ils deviennent
  éligibles pour des modèles et workloads adaptés.
- La roadmap ne dépend plus uniquement de futurs modèles type Fable/Mythos :
  BitNet est une variante plus concrète du même pari de densité.

## Déclencheur de réévaluation

- Disponibilité de modèles 1.58-bit open-weight de 7B-100B utiles en production
  sur les tâches Kappeler/JAIKIN.
- Serveur bitnet.cpp stable avec API streamable ou wrapper fiable derrière
  `Backend.Generate`.
- Bench local sur postes Kappeler : tok/s, RAM, énergie, bruit/thermique, impact
  utilisateur.
