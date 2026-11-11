# 11: Groupes

**What to build:** Jobierre range ses Stores dans des Groupes nommés, dans l'ordre qu'il veut, par glisser-déposer ou par formulaires sans JS. Un Groupe peut s'ouvrir en focus. Un Store retiré de la config alors qu'il était rangé reste à sa place comme Store disparu, jusqu'à ce qu'on le retire du Groupe ou qu'il réapparaisse (spec : stories 74 à 88, glossaire : Groupe, Store disparu).

**Blocked by:** 10 (Refonte de l'interface)

**Status:** ready-for-agent

- [ ] Les Groupes (nom unique, ordre), l'ordre des Stores dans chaque Groupe et les Stores disparus sont persistés de façon atomique dans le fichier d'état, et survivent au redémarrage.
- [ ] « + Groupe » (bandeau) crée un Groupe ; le menu ⋯ de l'en-tête le renomme ou le supprime. La suppression, après confirmation, renvoie ses Stores dans « Sans groupe » et efface ses Stores disparus. Rien ne touche à la config Plakar.
- [ ] Le nom réservé au focus de « Sans groupe » ne peut pas être pris par un Groupe.
- [ ] « Sans groupe » est toujours en dernier, ne se renomme, ne se supprime ni ne se glisse, et ne s'affiche que s'il contient au moins un Store. Un nouveau Store y va en dernier.
- [ ] Sans JS : dans le dépliage d'un Store, une liste déroulante « Groupe » et des boutons ↑/↓ ; dans l'en-tête d'un Groupe, de quoi le monter ou le descendre.
- [ ] Avec JS (fichier fait main, intégré au binaire, API HTML5 native) : glisser un Store dans un Groupe ou à une autre place, glisser un Groupe par sa poignée ⠿ ; chaque dépôt fait un `POST` puis recharge la page. Cela marche dans les deux vues : sections dans la vue détaillée, colonnes de tuiles dans la vue simple. Le JS n'insère jamais un nom de Store ou de Groupe comme du HTML.
- [ ] L'en-tête d'un Groupe montre le voyant du pire État de ses Stores et ses compteurs ; le chevron le replie, l'état replié est gardé dans `localStorage`.
- [ ] Un Store retiré de la config alors qu'il était dans un Groupe devient un Store disparu : grisé, « Store supprimé », bouton « Retirer du groupe ». Il garde sa place et son Seuil de retard, perd son port et son dernier Relevé, n'a pas d'État.
- [ ] Un Store qui réapparaît sous le même nom reprend la place et le Seuil de son Store disparu.
- [ ] Un Store « Sans groupe » retiré disparaît simplement.
- [ ] Le bandeau ajoute un compteur 👻 des Stores disparus, hors des compteurs d'État ; « Tout relever » les ignore.
- [ ] Un clic sur le nom d'un Groupe ouvre `/?groupe=<nom>` : seul ce Groupe s'affiche, avec « ← Tous les groupes ». Tout le bandeau (compteurs, 👻, bouton « Relever le groupe ») ne porte que sur les Stores du Groupe.
- [ ] Tests par le seam HTTP, via les formulaires : création, renommage, suppression, déplacement et ordre, Store disparu (`rm` puis `add`), focus et « Relever le groupe ».

## Comments
