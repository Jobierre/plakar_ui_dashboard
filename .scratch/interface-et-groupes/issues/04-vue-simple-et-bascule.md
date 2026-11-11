# 04: Vue simple et bascule

**What to build:** Une vue simple en tuiles, affichée par défaut pour le contrôle quotidien, où un clic sur une tuile ouvre un tiroir latéral avec les détails et les actions. Un sélecteur « Simple | Détaillée » dans le bandeau bascule d'une vue à l'autre, et le navigateur retient le choix (spec : stories 13, 14, 17 à 19, 68). Référence visuelle : variante D du prototype (`open "internal/web/prototype/ui-variantes.html?variant=D"`, branche `prototype/ui-variantes`).

**Blocked by:** 02 (Vue détaillée)

**Status:** resolved

- [x] Un petit fichier JS fait main, sans bibliothèque, est intégré au binaire, servi par le service et refusé à une identité hors de la liste.
- [x] Les deux vues partagent le même balisage ; une classe sur la page choisit la mise en page. Chaque Store n'apparaît qu'une fois dans le HTML.
- [x] Vue simple : une tuile par Store (bordure à la couleur de l'État, nom, âge du Dernier snapshot en gros, État, taille et Écart, erreur éventuelle). Un clic ouvre le tiroir latéral, qui est le dépliage du Store présenté en panneau.
- [x] La vue simple est la vue par défaut, y compris sans JavaScript.
- [x] Le sélecteur « Simple | Détaillée » est dans le bandeau ; le choix est retenu dans `localStorage`.
- [x] Le script n'insère jamais de nom comme du HTML (`textContent`, jamais `innerHTML`).
- [x] Tests par le seam HTTP : script servi et refusé selon l'identité, unicité de chaque Store. La bascule et le tiroir sont vérifiés à la main contre le prototype.

## Comments

**2026-11-11, verdict du prototype :** Jobierre garde la vue détaillée de la variante A et la vue simple de la variante B, avec une bascule (variante D). Vue simple par défaut, pour le contrôle rapide quotidien. JS accepté là où il simplifie. Variante C écartée.

**2026-11-11 :** implémenté. `<html class="vue-simple">` est rendu par le serveur ; `static/vue.js`, chargé dans `<head>` sans `defer`, applique la vue retenue (`localStorage`, clé `plakar-ui.vue`) avant le premier affichage, donc sans clignotement, même au rafraîchissement automatique. Le sélecteur est `hidden` sans JavaScript. La CSS de la liste est maintenant sous `.vue-detaillee`, celle des tuiles sous `.vue-simple` ; la bordure de la tuile vient d'une classe `bord-<voyant>` sur le `<details>`. Le tiroir est le `<details>` ouvert lui-même en panneau fixe : sa tuile en tête (nom, État, erreur, taille et Écart, ↻ et Explorer), qui le referme d'un clic, et le dépliage dessous. Il marche donc sans JavaScript, et s'ouvre seul après un échec (story 20). Le script ajoute un tiroir à la fois, Échap (hors saisie) et la fermeture des dépliages au passage en vue simple. L'heure seule du Relevé n'apparaît dans la vue simple que pendant un Relevé en cours (le dépliage donne la date). Vérifié dans Chrome contre la variante D : tuiles et tiroir à 1300 px, vue détaillée inchangée et retenue au rechargement, tiroir à 500 px, nom piégé `<b>piege</b>` affiché en texte.

Limites connues, à trancher :
- la tuile ouverte sort de la grille tant que le tiroir est ouvert (les suivantes se décalent), là où le prototype la gardait en place, surlignée ;
- le rafraîchissement automatique (ticket 03, toutes les 5 s pendant un Relevé) referme le tiroir ouvert à la main, comme il refermait déjà le dépliage ;
- pas de fond assombri ni de clic à l'extérieur pour fermer le tiroir.
