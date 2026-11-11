# 01: Feuille de style, voyant et bandeau

**What to build:** Le tableau de bord gagne sa feuille de style et un résumé d'un coup d'œil. Chaque Store porte un voyant coloré à côté du mot de son État. Un bandeau fixe en haut de page compte les Stores par État et montre l'identité connectée. C'est la balle traçante de la fonctionnalité : elle prouve toute la chaîne, d'un asset intégré au binaire et servi derrière l'Accès jusqu'aux tests HTTP (spec : `.scratch/interface-et-groupes/spec.md`, stories 1 à 6, 12, 67, 69).

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] Une feuille CSS faite main, sans framework ni CDN, est intégrée au binaire et servie par le service.
- [x] La feuille de style est refusée à une identité hors de la liste, comme le reste.
- [x] Thème clair et sombre selon `prefers-color-scheme`, sans bouton de bascule.
- [x] Chaque Store montre un voyant 🟢 À jour, 🔴 En retard ou 🟠 Injoignable, toujours accompagné du mot de l'État. Un Store sans Relevé terminé n'a pas de voyant d'État (« Premier Relevé »).
- [x] Un bandeau en haut de page, qui reste visible au défilement, compte les Stores À jour, En retard et Injoignables, et montre l'identité connectée.
- [x] Les routes existantes et les attributs `data-store`, `data-field`, `data-commande` restent valables ; les tests existants passent.
- [x] Tests par le seam HTTP : compteurs du bandeau (dont un Store sans Relevé, compté nulle part), mot de l'État présent avec le voyant, feuille de style servie à un Utilisateur autorisé et refusée sinon.

## Comments

## Comments

**2026-11-11 :** implémenté. Le voyant est une pastille CSS (vert, rouge, orange, comme le prototype) plutôt qu'un emoji ; elle est `aria-hidden`, le mot de l'État l'accompagne toujours. La feuille est servie sous `/static/style.css` par un serveur de fichiers générique sur le dossier `internal/web/static` intégré au binaire (le script des tickets suivants s'y ajoutera sans nouvelle route). Rendu visuel à vérifier à la main dans le navigateur (clair et sombre). « Premier Relevé » s'affiche à la place de l'État, mais n'est pas encore défini dans `GLOSSARY.md`.
