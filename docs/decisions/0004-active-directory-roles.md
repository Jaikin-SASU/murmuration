# ADR-0004 — Rôles de l'Active Directory

- Statut : Accepté
- Date : 2026-06-25

## Contexte

Le parc Kappeler est piloté par un serveur Windows AD. On veut s'appuyer dessus
plutôt que réinventer la gestion du parc.

## Décision

L'AD remplit quatre rôles, alimentant le scheduler :

1. **Inventaire des machines** — la liste des postes et leurs infos est lue
   depuis l'AD (LDAP) pour enrôler automatiquement les nodes.
2. **Déploiement des agents** — l'agent worker est poussé/mis à jour via GPO/MSI
   (Windows) ; équivalent pkg + profil pour Mac.
3. **Détection de présence** — savoir quelle session est ouverte/active sur
   chaque poste (l'agent local affine avec l'activité clavier/souris).
4. **Authentification de l'endpoint** *(défaut proposé)* — seuls les
   utilisateurs/groupes AD autorisés appellent l'API (LDAP/Kerberos → token).

Le **dispatch** du travail n'est PAS un rôle de l'AD : c'est le scheduler qui
décide, en consommant l'inventaire + la présence + la télémétrie temps réel.

## Conséquences

- Zéro double-saisie du parc ; cohérence avec l'existant Kappeler.
- Dépendance LDAP/Kerberos côté serveur ; à mocker en tests et en dev hors-AD.
- Les Mac non joints à l'AD nécessitent un chemin d'enrôlement alternatif
  (enrôlement manuel / MDM).

## Déclencheur de réévaluation

- Déploiement hors environnement AD (autre client OSS) → rendre l'AD optionnel
  derrière une interface `FleetSource` / `IdentityProvider`.
