# Contribuer à Murmuration

Merci de votre intérêt ! Murmuration est un projet open-source de JAIKIN SASU.

## Avant de commencer

- Lisez [`docs/DESIGN.md`](docs/DESIGN.md) et les [ADR](docs/decisions/README.md) :
  ils expliquent **pourquoi** l'architecture est ce qu'elle est. Beaucoup d'idées
  « évidentes » (éclater un gros modèle sur le parc) ont été écartées pour des raisons
  physiques documentées dans [`docs/PROBLEMS-AND-CONSTRAINTS.md`](docs/PROBLEMS-AND-CONSTRAINTS.md).
- Une grosse contribution ? Ouvrez d'abord une issue pour en discuter.

## Pré-requis

- Go 1.26+
- (optionnel) Ollama pour tester un node réel

## Boucle de développement

```bash
make test    # tests + couverture (objectif : ≥ 80 % par paquet)
make vet     # go vet doit être propre
make build   # compile bin/murmur
```

## Règles

- **Tests d'abord** quand c'est possible (TDD). Toute nouvelle logique vient avec ses tests.
- **Petits fichiers focalisés**, faible couplage. Une responsabilité par paquet.
- **Immutabilité** : on renvoie des copies, on ne mute pas l'état partagé (voir
  `capability.Node.Clone`).
- **Pas de dépendance externe sans discussion** : la v1 est volontairement stdlib-only.
- Nouveau moteur d'inférence ? Implémentez l'interface `backend.Backend` — ne touchez
  pas au scheduler (c'est tout l'intérêt de l'abstraction, ADR-0006).
- Décision structurante ? Ajoutez un ADR dans `docs/decisions/`.

## Commits & PR

- Commits clairs, en français ou anglais.
- Une PR = un sujet. Décrivez le quoi et le pourquoi.
- CI verte (vet + tests) obligatoire avant merge.

## Licence

En contribuant, vous acceptez que votre code soit distribué sous licence [MIT](LICENSE).
