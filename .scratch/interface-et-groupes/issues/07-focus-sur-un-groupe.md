# 07: Focus sur un Groupe

**What to build:** Un clic sur le nom d'un Groupe n'affiche plus que ce Groupe, à une adresse qu'on peut mettre en favori. En focus, le bandeau ne porte que sur les Stores du Groupe, et son bouton relève le Groupe (spec : stories 51 à 57).

**Blocked by:** 03 (Tout relever et rafraîchissement automatique), 05 (Groupes, créer et ranger)

**Status:** resolved

- [x] `/?groupe=<nom>` n'affiche que ce Groupe, avec un lien « ← Tous les groupes ». Le nom du Groupe dans l'en-tête mène à ce focus.
- [x] Un mot réservé ouvre le focus de « Sans groupe ».
- [x] En focus, les compteurs du bandeau ne portent que sur les Stores du Groupe. Le bouton s'appelle « Relever le groupe » et ne relève que ces Stores, en sautant ceux déjà en cours.
- [x] Le rafraîchissement automatique ne tient compte que des Relevés en cours des Stores affichés.
- [x] Le focus d'un Groupe inconnu (renommé, supprimé, adresse fausse) affiche un message et le lien « ← Tous les groupes ».
- [x] Tests par le seam HTTP : contenu du focus, focus de « Sans groupe », compteurs en focus, Relever le groupe, focus inconnu.

## Comments

- Implémenté : `/?groupe=<nom>` (ou `_sans`) filtre les sections et le bandeau ; `POST /releve` accepte `groupe` et revient sur le focus ; `Catalogue.Membres` donne les Stores d'un Groupe. Sur un focus inconnu, le bouton du bandeau est masqué.
- Décidé (domain-modeling) : le Focus est une page de consultation. Les autres actions faites en focus (Seuil, ↻, rangement, ordre, renommage) ramènent volontairement à `/`. Les libellés « Relever le groupe » et « ← Tous les groupes » gardent la minuscule.
- Reste ouvert : le compteur 👻 (ticket 08) devra être filtré par le focus.
