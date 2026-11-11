# 03: Tout relever et rafraîchissement automatique

**What to build:** Un bouton « Tout relever » dans le bandeau lance un Relevé de chaque Store, sans jamais chevaucher un Relevé en cours. Tant qu'un Relevé est en cours, la page se rafraîchit seule, pour que Jobierre voie les résultats arriver (spec : stories 22 à 27).

**Blocked by:** 01 (Feuille de style, voyant et bandeau)

**Status:** ready-for-agent

- [ ] Une action `POST` relève chaque Store, en sautant ceux dont un Relevé est déjà en cours, puis redirige vers la page. Elle réutilise le déclenchement par Store et sa garantie de non-chevauchement.
- [ ] Le bouton « Tout relever » du bandeau est désactivé quand tous les Stores ont déjà un Relevé en cours.
- [ ] Le bouton ↻ par Store reste disponible.
- [ ] La page émet `<meta http-equiv="refresh" content="5">` seulement tant qu'au moins un Relevé est en cours.
- [ ] L'action est refusée à une identité hors de la liste.
- [ ] Tests par le seam HTTP : chaque Store relevé une fois, Store déjà en cours non relancé, bouton désactivé, présence puis absence du rafraîchissement, refus de l'Accès.

## Comments
