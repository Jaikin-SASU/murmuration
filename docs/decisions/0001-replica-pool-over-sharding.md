# ADR-0001 — Pool de répliques comme substrat, pas l'agrégation mémoire

- Statut : Accepté
- Date : 2026-06-25

## Contexte

Voir [P1](../PROBLEMS-AND-CONSTRAINTS.md#p1) et [P2](../PROBLEMS-AND-CONSTRAINTS.md#p2).
On ne peut pas agréger la RAM/GPU de 30 postes en « une grosse machine », et
éclater un très gros modèle sur le parc plafonne à quelques tok/s.

## Décision

Le **substrat durable** du cluster est un **pool de répliques** : chaque poste
capable fait tourner une **copie entière** d'un modèle qui tient dans sa VRAM/RAM
(7B–14B quantifié, jusqu'à ~30B sur les gros GPU). Une gateway répartit les
requêtes sur les répliques.

L'éclatement de modèle (pipeline) est une **capacité secondaire** réservée au
mode nuit, batch uniquement, sur 2-4 nœuds stables (voir
[ADR-0003](0003-engines-ollama-llamacpp.md)).

## Conséquences

- Robustesse : un poste tombe = on perd une réplique, pas l'inférence.
- Débit agrégé élevé + latence faible (pas de mur de latence série).
- Limite assumée : un modèle plus gros qu'un poste ne tourne **pas** en
  interactif sur le parc → il reste sur le serveur JAIKIN.
- Les gros modèles vivent sur JAIKIN ; le parc Kappeler sert les modèles
  petits/moyens, les tâches parallélisables et l'overflow.

## Déclencheur de réévaluation

- LAN ≥ 10 GbE / Thunderbolt mesh → reconsidérer l'éclatement interactif.
- Densité des modèles : si un 14B atteint la qualité d'un 200B actuel, le pool
  fait tourner du « gros » sans changement d'architecture → **c'est le 10x**.
