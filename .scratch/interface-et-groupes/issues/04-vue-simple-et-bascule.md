# 04: Vue simple et bascule

**What to build:** Une vue simple en tuiles, affichée par défaut pour le contrôle quotidien, où un clic sur une tuile ouvre un tiroir latéral avec les détails et les actions. Un sélecteur « Simple | Détaillée » dans le bandeau bascule d'une vue à l'autre, et le navigateur retient le choix (spec : stories 13, 14, 17 à 19, 68). Référence visuelle : variante D du prototype (`open "internal/web/prototype/ui-variantes.html?variant=D"`, branche `prototype/ui-variantes`).

**Blocked by:** 02 (Vue détaillée)

**Status:** ready-for-agent

- [ ] Un petit fichier JS fait main, sans bibliothèque, est intégré au binaire, servi par le service et refusé à une identité hors de la liste.
- [ ] Les deux vues partagent le même balisage ; une classe sur la page choisit la mise en page. Chaque Store n'apparaît qu'une fois dans le HTML.
- [ ] Vue simple : une tuile par Store (bordure à la couleur de l'État, nom, âge du Dernier snapshot en gros, État, taille et Écart, erreur éventuelle). Un clic ouvre le tiroir latéral, qui est le dépliage du Store présenté en panneau.
- [ ] La vue simple est la vue par défaut, y compris sans JavaScript.
- [ ] Le sélecteur « Simple | Détaillée » est dans le bandeau ; le choix est retenu dans `localStorage`.
- [ ] Le script n'insère jamais de nom comme du HTML (`textContent`, jamais `innerHTML`).
- [ ] Tests par le seam HTTP : script servi et refusé selon l'identité, unicité de chaque Store. La bascule et le tiroir sont vérifiés à la main contre le prototype.

## Comments

**2026-11-11, verdict du prototype :** Jobierre garde la vue détaillée de la variante A et la vue simple de la variante B, avec une bascule (variante D). Vue simple par défaut, pour le contrôle rapide quotidien. JS accepté là où il simplifie. Variante C écartée.
