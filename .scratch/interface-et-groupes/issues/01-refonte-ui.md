# 01: Refonte de l'interface

**What to build:** Le tableau de bord en HTML brut (21 colonnes, sans CSS) devient une interface lisible sur l'écran du Mac : une ligne compacte par Store avec un voyant d'État, un bandeau de résumé avec « Tout relever », un dépliage pour les détails et la Commande de Store rangée dans une section repliable. Deux vues au choix, simple (tuiles, par défaut) et détaillée (liste dense). Rendu côté serveur, CSS et petit JS faits main intégrés au binaire (spec : `.scratch/interface-et-groupes/spec.md`, stories 1 à 30 et 67 à 69).

**Prototype de référence :** branche `prototype/ui-variantes`, `open "internal/web/prototype/ui-variantes.html?variant=D"`. Le verdict est dans les Comments.

**Blocked by:** aucun

**Status:** ready-for-agent

- [ ] Une feuille CSS et un petit fichier JS faits main sont intégrés au binaire (`//go:embed`), servis sans CDN, sans bibliothèque ni étape de build ; thème clair et sombre selon `prefers-color-scheme`. Le JS n'insère jamais de nom comme du HTML (`textContent`).
- [ ] Chaque Store montre un voyant 🟢 À jour, 🔴 En retard ou 🟠 Injoignable, toujours accompagné du mot de l'État.
- [ ] Un bandeau en haut compte les Stores par État, montre l'utilisateur connecté et porte le bouton « Tout relever ».
- [ ] `POST /releves` lance un Relevé de chaque Store, sauf ceux dont le Relevé est déjà en cours, puis redirige vers `/`. Le bouton est désactivé si tous les Stores sont en cours.
- [ ] La page émet `<meta http-equiv="refresh" content="5">` seulement tant qu'au moins un Relevé est en cours.
- [ ] Vue simple, affichée par défaut : une tuile par Store (bordure à la couleur de l'État, nom, âge en gros, État, taille et Écart, erreur éventuelle). Un clic ouvre un tiroir latéral avec les détails et les actions.
- [ ] Bascule « Simple | Détaillée » dans le bandeau, retenue par le navigateur (`localStorage`). Si possible, même balisage pour les deux vues et une classe CSS qui change, pour que chaque Store n'apparaisse qu'une fois.
- [ ] Vue détaillée, ligne repliée : voyant et État, nom, âge du Dernier snapshot, taille et Écart, heure du dernier Relevé (ou « Relevé en cours »), ↻, Explorer ou lien et Arrêter.
- [ ] Dépliage (`<details>`) : date exacte, ID, durée et Écart, source, tags, Seuil de retard et son formulaire, message de port. Il s'ouvre seul après une erreur de Seuil ou d'Exploration sur ce Store.
- [ ] Un Store Injoignable affiche son erreur sans dépliage, et ses dernières valeurs connues sont atténuées.
- [ ] L'Écart s'affiche en texte neutre, signé, avec ↑/↓ et la valeur précédente en infobulle, sans couleur d'alerte.
- [ ] La Commande de Store est dans une section repliable « Gérer les Stores », ouverte d'elle-même quand un résultat (refus, erreur, sortie) s'affiche ; stdout et stderr en blocs façon terminal.
- [ ] Sous 700 px de large, chaque ligne s'empile en carte.
- [ ] L'ordre par nom est conservé ; les routes existantes et les attributs `data-store`, `data-field`, `data-commande` restent valables, et les tests existants passent.
- [ ] Nouveaux tests par le seam HTTP : « Tout relever » (Stores en cours ignorés), présence ou absence du rafraîchissement, compteurs du bandeau.

## Comments

**2026-11-11, verdict du prototype :** Jobierre garde la vue détaillée de la variante A (liste dense) et la vue simple de la variante B (tuiles avec tiroir), avec une bascule entre les deux (variante D). La vue simple s'affiche par défaut, pour le contrôle rapide quotidien. Le JS est accepté là où il simplifie. Variante C (maître-détail) écartée.
