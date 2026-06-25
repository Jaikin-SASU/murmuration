# ADR-0008 — Transport HTTP/JSON pour la v1 (gRPC différé)

- Statut : Accepté
- Date : 2026-06-25

## Contexte

Le design initial ([DESIGN](../DESIGN.md)) évoquait gRPC bidirectionnel pour
agent↔serveur. gRPC impose `protoc` + plugins de génération et des dépendances
externes, ce qui alourdit le build, la contribution OSS et la reproductibilité.

## Décision

La v1 utilise **HTTP/JSON** (stdlib Go uniquement) pour les deux plans :

- **Plan public** (clients → gateway) : REST OpenAI-compatible, streaming SSE.
- **Plan de contrôle** (agent → serveur) : endpoints `internal/v1` JSON
  (`register`, `heartbeat`). Le heartbeat porte la télémétrie + la présence.

Zéro dépendance externe → `go build ./...` et `go test ./...` fonctionnent hors
ligne, sur tout OS.

## Conséquences

- Mise en route et contribution triviales (aucun codegen).
- Pas de push serveur→agent natif : l'agent **poll** le serveur au heartbeat pour
  récupérer ses ordres (scale-up/down). Acceptable aux cadences visées (2-3 s).
- Le passage à gRPC reste possible derrière les mêmes interfaces si le besoin de
  streaming bidirectionnel à faible latence apparaît.

## Déclencheur de réévaluation

- Besoin de commandes serveur→agent en temps réel (< 1 s) ou de très nombreux
  nodes → reconsidérer gRPC / WebSocket.
