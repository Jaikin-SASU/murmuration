# ADR-0007 — Sharding optimisé par l'IA (piste de recherche)

- Statut : Proposé (recherche / spéculatif)
- Date : 2026-06-25

## Contexte

[ADR-0006](0006-future-proof-pluggable-backends.md) parie sur la densité des
modèles pour le 10x « gratuit ». L'utilisateur veut un **second levier de 10x** :
rendre l'éclatement (sharding) viable même sur réseau lent, en confiant son
optimisation à des modèles futurs (Fable, « Mythos »).

L'intuition : le sharding sur LAN lent est un **problème d'optimisation dur**
(quelle couche sur quel node, comment masquer la latence série, comment
prefetch/spéculer), pas un mur physique absolu. Les marges identifiées dans le
journal le confirment :
- bande passante quasi non-limitante en pipeline ([P2](../PROBLEMS-AND-CONSTRAINTS.md#p2))
  → le vrai ennemi est la **latence série** et la **compute par node** ;
- des techniques connues attaquent déjà ça : quantification 8-bit des activations
  (Petals), ring + prefetch (prima.cpp), partition pondérée RAM (exo).

## Décision (piste, pas engagement v1)

Garder le sharding comme **track de recherche** branché sur l'interface `Backend`,
avec pour objectif explicite que l'**optimiseur de partition/ordonnancement soit
piloté par modèle**. Idées à explorer (sans s'y engager pour v1) :

- **Planificateur de partition appris** : un modèle ingère le graphe du parc
  (compute par node, RTT par lien, VRAM, présence) et produit le plan de découpe
  + placement qui maximise les tok/s. Ré-évalué dynamiquement.
- **Décodage spéculatif distribué** : un petit modèle « draft » sur un node rapide
  propose des tokens, vérifiés en lot par le modèle shardé → amortit la latence
  série (le coût dominant, [P2](../PROBLEMS-AND-CONSTRAINTS.md#p2)).
- **Masquage de latence par prefetch/pipelining agressif** appris par node.
- **Compression d'activations adaptative** pilotée par modèle selon le lien.

Mesure de succès : tok/s sur un groupe nuit 2-4 nœuds, avant/après, sur le même
matériel et le même LAN.

## Conséquences

- Aucune dépendance v1 : le pool de répliques ([ADR-0001](0001-replica-pool-over-sharding.md))
  reste le substrat. Cette piste ne doit pas retarder le cœur.
- L'interface `Backend` + le capability registry fournissent déjà les entrées
  (graphe du parc) dont un planificateur appris a besoin → rien à réécrire pour
  expérimenter.
- Risque : surface de recherche large ; à cadrer en expériences mesurables, pas
  en refonte.

## Déclencheur de réévaluation

- Disponibilité d'un modèle capable (Fable/Mythos…) assez bon pour planifier/
  spéculer en temps réel → promouvoir la piste de « recherche » à « backend ».
- Si la latence série reste indomptable malgré ces techniques → archiver et
  rester sur le pari densité.
