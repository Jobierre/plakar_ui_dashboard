# 06: Groupes, ordre, renommage et suppression

**What to build:** Jobierre choisit l'ordre de ses Groupes et celui des Stores dans chaque Groupe, renomme un Groupe et le supprime. L'ordre survit au redémarrage (spec : stories 37, 39, 40, 45 pour ↑/↓, 46, 47, 50 pour le menu ⋯).

**Blocked by:** 05 (Groupes, créer et ranger)

**Status:** resolved

- [x] Des boutons ↑/↓ dans le dépliage d'un Store le déplacent dans son Groupe, « Sans groupe » compris.
- [x] Des boutons ↑/↓ sur l'en-tête d'un Groupe le déplacent parmi les Groupes. « Sans groupe » reste en dernier et ne se déplace pas.
- [x] Le rangement d'un Store accepte une position (le Store avant lequel l'insérer), pour que le glisser-déposer du ticket 09 réutilise la même action. Le déplacement d'un Groupe accepte de même le Groupe avant lequel l'insérer.
- [x] Le menu ⋯ d'un Groupe propose Renommer (mêmes règles de nom qu'à la création) et Supprimer, après confirmation. La suppression renvoie ses Stores à la fin de « Sans groupe », dans leur ordre.
- [x] « Sans groupe » ne se renomme ni ne se supprime.
- [x] Une action sur un Groupe ou un Store inconnu renvoie à la page avec un message, sans rien modifier. Les actions sont refusées à une identité hors de la liste.
- [x] Tests par le seam HTTP : ordre des Stores et des Groupes (avec ↑/↓ et avec une position), renommage et ses refus, suppression, cas inconnus, persistance de l'ordre après redémarrage, refus de l'Accès.

## Comments

Implémenté le 2026-11-11. Contrat des routes (toutes en POST, puis redirection vers `/`) :

- `POST /stores/{store}/ordre`, `sens=haut|bas` : ↑/↓ d'un Store dans son Groupe ou dans « Sans groupe ».
- `POST /stores/{store}/groupe`, `groupe=<nom|_sans>`, `avant=<store>` en option. Si `avant` est présent, il donne une position explicite ; vide, il place le Store à la fin. Pour le ticket 09, il faut donc toujours envoyer `avant`. Sans `avant` (liste déroulante), un Store déjà dans ce Groupe garde sa place.
- `POST /groupes/{groupe}/ordre`, `sens=haut|bas` ou `avant=<groupe>`. Un `avant` vide ou égal à `_sans` place le Groupe en dernier.
- `POST /groupes/{groupe}/nom`, `nom=…` : mêmes règles qu'à la création. Le nom actuel est accepté sans effet.
- `POST /groupes/{groupe}/suppression` : la confirmation passe par un `<details>` dans le menu ⋯, sans JavaScript.

Les noms vont dans l'adresse échappés par `url.PathEscape` (fonction `chemin` du template), pour toutes les routes de Store et de Groupe. Un refus de rangement rouvre le dépliage du Store visé.
