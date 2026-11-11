# 08: Stores disparus

**What to build:** Un Store retiré de la config Plakar alors qu'il était rangé dans un Groupe reste à sa place comme Store disparu. Un `store rm` suivi d'un `store add` du même nom ne défait plus le rangement. Jobierre fait le ménage quand il le décide (spec : stories 58 à 66 ; glossaire : Store disparu).

**Blocked by:** 03 (Tout relever et rafraîchissement automatique), 06 (Groupes, ordre, renommage et suppression)

**Status:** resolved

- [x] **Changement du Catalogue** : il ne jette plus l'état d'un Store absent de la config s'il est membre d'un Groupe. Son Seuil de retard est gardé, son port d'Exploration libéré, son dernier Relevé oublié, et il n'est plus proposé au planificateur. Un Store « Sans groupe » absent de la config est oublié comme avant.
- [x] Un Store disparu s'affiche à sa place, grisé, sans État ni valeurs, avec « Store supprimé » et un bouton « Retirer du groupe », qui l'oublie (Seuil compris).
- [x] Un Store qui réapparaît dans la config sous le même nom reprend la place et le Seuil de son Store disparu, et obtient un nouveau port et un premier Relevé.
- [x] Le bandeau montre un compteur 👻 des Stores disparus, à part des compteurs d'État ; en focus, il ne compte que ceux du Groupe.
- [x] Tout relever et Relever le groupe ignorent les Stores disparus.
- [x] Supprimer un Groupe oublie ses Stores disparus.
- [x] Les Stores disparus survivent au redémarrage.
- [x] Tests par le seam HTTP, avec le faux `plakar` qui change sa liste de Stores puis une Commande de Store : `rm` d'un Store rangé, affichage, port libéré, `add` du même nom, « Retirer du groupe », `rm` d'un Store « Sans groupe », compteur 👻, Tout relever, suppression du Groupe, redémarrage.

## Comments

- Implémenté : un membre de Groupe absent de la config est un Store disparu (`Entry.Disparu`) ; seul son Seuil est gardé, dans le fichier d'état. `POST /stores/<store>/retrait` (« Retirer du groupe ») l'oublie ; un Store de la config y est refusé (400), un nom inconnu aussi (404). Tout Seuil qui n'est plus ni à un Store ni à un Store disparu est oublié à chaque changement du rangement, avec annulation si l'enregistrement échoue.
- Décidé (Jobierre) : le compteur 👻 du bandeau est toujours affiché, même à 0. Un Store disparu se déplace par ↑/↓ sur sa ligne (sa place est la sienne), mais ne se range pas dans un autre Groupe.
- Le compteur 👻 en focus (reste ouvert du ticket 07) est fait.
