# 02: Vue détaillée

**What to build:** Le tableau de 21 colonnes devient une liste dense : une ligne compacte par Store, et un dépliage pour le reste. L'erreur d'un Store Injoignable se voit sans déplier, l'Écart s'affiche en texte neutre, et la Commande de Store est rangée dans une section repliable (spec : stories 7 à 11, 15, 16, 20, 21, 28 à 30). Référence visuelle : variante A du prototype (branche `prototype/ui-variantes`, `open "internal/web/prototype/ui-variantes.html?variant=A"`).

**Blocked by:** 01 (Feuille de style, voyant et bandeau)

**Status:** ready-for-agent

- [ ] Ligne repliée : voyant et État, nom, âge du Dernier snapshot, taille et Écart de taille, heure du dernier Relevé ou « Relevé en cours », ↻, Explorer ou lien d'Exploration et Arrêter.
- [ ] Dépliage : date exacte, ID court, durée et Écart de durée, chemin source, tags, Seuil de retard avec son formulaire, message de port.
- [ ] Le dépliage d'un Store s'ouvre de lui-même après une erreur de Seuil ou d'Exploration sur ce Store.
- [ ] L'erreur d'un Store Injoignable est visible dans la ligne repliée ; ses dernières valeurs connues sont atténuées.
- [ ] L'Écart est en texte neutre et signé, avec ↑, ↓ ou →, la valeur du Snapshot précédent en infobulle, et aucune couleur d'alerte.
- [ ] La Commande de Store est dans une section « Gérer les Stores », fermée par défaut et ouverte d'elle-même quand un résultat s'affiche ; stdout et stderr en blocs façon terminal.
- [ ] Sous 700 px de large, chaque ligne s'empile en carte.
- [ ] Chaque Store n'apparaît qu'une fois dans la page.
- [ ] Tests par le seam HTTP : contenu de la ligne et du dépliage par `data-field`, dépliage ouvert après une erreur de Seuil, section « Gérer les Stores » ouverte après une Commande de Store, unicité de chaque Store.

## Comments
