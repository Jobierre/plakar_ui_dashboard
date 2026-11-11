# 10: Refonte de l'interface

**What to build:** Le tableau de bord en HTML brut (21 colonnes, sans CSS) devient une interface lisible sur l'écran du Mac : une ligne compacte par Store avec un voyant d'État, un bandeau de résumé avec « Tout relever », un dépliage pour les détails et la Commande de Store rangée dans une section repliable. Rendu côté serveur, CSS fait main intégré au binaire, aucun JavaScript (spec : stories 63 à 73, « Interface (décidée le 2026-11-11) »).

**Blocked by:** aucun

**Status:** ready-for-agent

- [ ] Une feuille CSS faite main est intégrée au binaire (`//go:embed`), servie sans CDN ni étape de build ; thème clair et sombre selon `prefers-color-scheme`.
- [ ] Chaque Store montre un voyant 🟢 À jour, 🔴 En retard ou 🟠 Injoignable, toujours accompagné du mot de l'État.
- [ ] Un bandeau en haut compte les Stores par État, montre l'utilisateur connecté et porte le bouton « Tout relever ».
- [ ] `POST /releves` lance un Relevé de chaque Store, sauf ceux dont le Relevé est déjà en cours, puis redirige vers `/`. Le bouton est désactivé si tous les Stores sont en cours.
- [ ] La page émet `<meta http-equiv="refresh" content="5">` seulement tant qu'au moins un Relevé est en cours.
- [ ] Ligne repliée : voyant et État, nom, âge du Dernier snapshot, taille et Écart, heure du dernier Relevé (ou « Relevé en cours »), ↻, Explorer ou lien et Arrêter.
- [ ] Dépliage (`<details>`) : date exacte, ID, durée et Écart, source, tags, Seuil de retard et son formulaire, message de port. Il s'ouvre seul après une erreur de Seuil ou d'Exploration sur ce Store.
- [ ] Un Store Injoignable affiche son erreur sans dépliage, et ses dernières valeurs connues sont atténuées.
- [ ] L'Écart s'affiche en texte neutre, signé, avec ↑/↓ et la valeur précédente en infobulle, sans couleur d'alerte.
- [ ] La Commande de Store est dans une section repliable « Gérer les Stores », ouverte d'elle-même quand un résultat (refus, erreur, sortie) s'affiche ; stdout et stderr en blocs façon terminal.
- [ ] Sous 700 px de large, chaque ligne s'empile en carte.
- [ ] L'ordre par nom est conservé ; les routes existantes et les attributs `data-store`, `data-field`, `data-commande` restent valables, et les tests existants passent.
- [ ] Nouveaux tests par le seam HTTP : « Tout relever » (Stores en cours ignorés), présence ou absence du rafraîchissement, compteurs du bandeau.

## Comments
