# 09: Glisser-déposer et repli des Groupes

**What to build:** Jobierre range ses Stores et ordonne ses Groupes à la souris, par glisser-déposer, dans les deux vues. Il replie les Groupes qui vont bien, et son navigateur s'en souvient (spec : stories 42 à 44, 49, 50, 68).

**Blocked by:** 04 (Vue simple et bascule), 06 (Groupes, ordre, renommage et suppression)

**Status:** resolved

- [x] Glisser un Store dans un autre Groupe ou à une autre place de son Groupe, dans la vue détaillée (sections) comme dans la vue simple (colonnes de tuiles). Chaque dépôt envoie le même `POST` que le rangement avec position du ticket 06, puis recharge la page.
- [x] Glisser un Groupe par sa poignée ⠿ change l'ordre des Groupes ; « Sans groupe » ne se glisse pas et reste en dernier.
- [x] Le chevron ▾ replie ou déplie un Groupe ; l'état replié est retenu dans `localStorage`, pas par le service.
- [x] Les quatre gestes de l'en-tête sont distincts : poignée pour glisser, nom pour le focus, chevron pour replier, menu ⋯ pour renommer ou supprimer.
- [x] Le script utilise l'API HTML5 native, sans bibliothèque, et n'insère jamais de nom comme du HTML.
- [x] Vérification à la main dans le navigateur, contre le prototype (variante D) : pas de test automatique du JavaScript.

## Comments

- Implémenté, sans changement côté service : les routes acceptaient déjà le champ `avant` (ticket 06). Le script envoie le formulaire de rangement déjà présent dans la page (la liste « Groupe » du Store, les ↑/↓ du Groupe), avec `avant` en plus. Il ne réécrit donc aucune route. La poignée ⠿ et le chevron ▾ sont dans le template, en `hidden`, et le script les affiche. Le `draggable` des Stores est posé par le script. « Sans groupe » porte `data-fixe`.
- Le sens du dépôt suit la mise en page, sans cas codé par vue : en grille de plusieurs colonnes (tuiles, colonnes de Groupes), la moitié gauche ou droite décide ; sinon, la moitié haute ou basse. Un dépôt qui laisserait l'élément à sa place n'envoie rien.
- Un Store disparu ne se glisse pas, car `Placer` le refuse : il se déplace toujours par ↑/↓. On peut en revanche poser un Store avant ou après lui.
- Repli : clé `plakar-ui.replis` dans `localStorage`. Un Groupe qui contient un Store ouvert par le service (une action qui vient d'échouer) ne se replie pas. Les clés des Groupes renommés ou supprimés ne sont pas purgées, parce que la page ne montre pas tous les Groupes (focus, « Sans groupe » vide caché).
- Vérifié dans Chrome contre un faux `plakar` : Store vers un autre Groupe, avant et après un Store, dans les deux vues ; Groupe avant un autre et sur « Sans groupe » ; repli retenu après rechargement ; Groupe au nom piégé (HTML, `/`, accent) sans injection.
- Reste ouvert, à trancher :
  - « Sans groupe » vide n'est pas affiché, donc on ne peut pas y déposer un Store à la souris (la liste « Groupe » reste disponible).
  - Comme pour les ↑/↓, un dépôt fait en focus renvoie à `/` et sort du focus.
