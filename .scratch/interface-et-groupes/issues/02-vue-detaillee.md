# 02: Vue détaillée

**What to build:** Le tableau de 21 colonnes devient une liste dense : une ligne compacte par Store, et un dépliage pour le reste. L'erreur d'un Store Injoignable se voit sans déplier, l'Écart s'affiche en texte neutre, et la Commande de Store est rangée dans une section repliable (spec : stories 7 à 11, 15, 16, 20, 21, 28 à 30). Référence visuelle : variante A du prototype (branche `prototype/ui-variantes`, `open "internal/web/prototype/ui-variantes.html?variant=A"`).

**Blocked by:** 01 (Feuille de style, voyant et bandeau)

**Status:** resolved

- [x] Ligne repliée : voyant et État, nom, âge du Dernier snapshot, taille et Écart de taille, heure du dernier Relevé ou « Relevé en cours », ↻, Explorer ou lien d'Exploration et Arrêter.
- [x] Dépliage : date exacte, ID court, durée et Écart de durée, chemin source, tags, Seuil de retard avec son formulaire, message de port.
- [x] Le dépliage d'un Store s'ouvre de lui-même après une erreur de Seuil ou d'Exploration sur ce Store.
- [x] L'erreur d'un Store Injoignable est visible dans la ligne repliée ; ses dernières valeurs connues sont atténuées.
- [x] L'Écart est en texte neutre et signé, avec ↑, ↓ ou →, la valeur du Snapshot précédent en infobulle, et aucune couleur d'alerte.
- [x] La Commande de Store est dans une section « Gérer les Stores », fermée par défaut et ouverte d'elle-même quand un résultat s'affiche ; stdout et stderr en blocs façon terminal.
- [x] Sous 700 px de large, chaque ligne s'empile en carte.
- [x] Chaque Store n'apparaît qu'une fois dans la page.
- [x] Tests par le seam HTTP : contenu de la ligne et du dépliage par `data-field`, dépliage ouvert après une erreur de Seuil, section « Gérer les Stores » ouverte après une Commande de Store, unicité de chaque Store.

## Comments

**2026-11-11 :** implémenté. Chaque Store est un `<details>` : la ligne repliée est son `<summary>`, le dépliage le reste. Les boutons de la ligne (↻, Explorer, Arrêter) y sont rattachés par l'attribut `form="…"` à des formulaires vides placés juste après, car un `<form>` ne peut pas vivre dans un `<summary>` ; tout marche sans JavaScript. La ligne montre l'heure seule du dernier Relevé (`data-field="releve-heure"`, date complète en infobulle). La date complète (`releve`) est dans le dépliage, lisible même pendant un Relevé en cours. Le lien d'Exploration garde l'URL comme texte (les tests la lisent là), tronqué par la CSS : le prototype affichait « Ouvrir ↗ », à revoir si ça gêne. L'atténuation d'un Store Injoignable porte sur la ligne et le dépliage. Rendu vérifié dans Chrome : clair à 1300 px, sombre à 390 px (empilement en carte), section « Gérer les Stores » ouverte sur un résultat. Limite connue : boutons et lien dans un `<summary>` forment du contenu interactif imbriqué, comme dans le prototype, et un lecteur d'écran peut les aplatir.
