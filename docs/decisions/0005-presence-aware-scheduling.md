# ADR-0005 — Scheduling conscient de la présence

- Statut : Accepté
- Date : 2026-06-25

## Contexte

Voir [P4](../PROBLEMS-AND-CONSTRAINTS.md#p4). Le poste appartient d'abord à
l'utilisateur assis devant. Et on veut privilégier les gros GPU.

## Décision

### Score de capacité (gros GPU d'abord)

```
score = poids_VRAM_libre × tier_GPU × dispo_présence − pénalité_réseau − charge
```
`tier_GPU` : GPU dédié costaud ≫ GPU moyen ≫ Apple Silicon (Metal) ≫ CPU-only.
Une requête va au node au meilleur score ayant **déjà** le modèle chargé, sinon
on en chauffe un (scale-up).

### Trois niveaux de présence (garde-fou local sur l'agent)

| État | Politique |
|---|---|
| **Actif** (input < ~5 min, ou plein écran/jeu) | Node retiré du pool pour les nouvelles requêtes ; requête en cours drainée ; mode nuit interdit. |
| **Idle** (pas d'input, session ouverte) | Disponible mais plafonné (≤ ~60 % VRAM, priorité basse). |
| **Libre** (session fermée / hors heures / nuit) | Pleine puissance : pool + pipeline RPC éligibles. |

Si l'utilisateur tape pendant une inférence : l'agent **throttle** le process
moteur et signale au scheduler de ne plus l'élire.

### Auto-scaling

- Profondeur de file + latence par modèle pilotent scale-up / scale-down.
- Node inutilisé X min → déchargement du modèle (libère la VRAM).
- Fenêtre nuit configurable → bascule full power + pipeline pour modèles un cran
  trop gros.

### Robustesse

- Heartbeat manqué → node sorti du pool, requêtes en vol réémises (idempotence
  gateway).
- LAN faible → forte pénalité réseau : on garde une requête sur un seul node ;
  l'éclatement multi-nodes n'est tenté que la nuit, en pipeline.

## Conséquences

- L'utilisateur ne subit jamais de ralentissement perceptible (objectif n°1).
- Capacité « jour » variable et imprévisible → la file batch absorbe le reste.

## Déclencheur de réévaluation

- Postes dédiés (sans utilisateur) → niveau « Libre » permanent, score simplifié.
