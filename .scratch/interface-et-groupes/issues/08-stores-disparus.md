# 08: Stores disparus

**What to build:** Un Store retiré de la config Plakar alors qu'il était rangé dans un Groupe reste à sa place comme Store disparu. Un `store rm` suivi d'un `store add` du même nom ne défait plus le rangement. Jobierre fait le ménage quand il le décide (spec : stories 58 à 66 ; glossaire : Store disparu).

**Blocked by:** 03 (Tout relever et rafraîchissement automatique), 06 (Groupes, ordre, renommage et suppression)

**Status:** ready-for-agent

- [ ] **Changement du Catalogue** : il ne jette plus l'état d'un Store absent de la config s'il est membre d'un Groupe. Son Seuil de retard est gardé, son port d'Exploration libéré, son dernier Relevé oublié, et il n'est plus proposé au planificateur. Un Store « Sans groupe » absent de la config est oublié comme avant.
- [ ] Un Store disparu s'affiche à sa place, grisé, sans État ni valeurs, avec « Store supprimé » et un bouton « Retirer du groupe », qui l'oublie (Seuil compris).
- [ ] Un Store qui réapparaît dans la config sous le même nom reprend la place et le Seuil de son Store disparu, et obtient un nouveau port et un premier Relevé.
- [ ] Le bandeau montre un compteur 👻 des Stores disparus, à part des compteurs d'État ; en focus, il ne compte que ceux du Groupe.
- [ ] Tout relever et Relever le groupe ignorent les Stores disparus.
- [ ] Supprimer un Groupe oublie ses Stores disparus.
- [ ] Les Stores disparus survivent au redémarrage.
- [ ] Tests par le seam HTTP, avec le faux `plakar` qui change sa liste de Stores puis une Commande de Store : `rm` d'un Store rangé, affichage, port libéré, `add` du même nom, « Retirer du groupe », `rm` d'un Store « Sans groupe », compteur 👻, Tout relever, suppression du Groupe, redémarrage.

## Comments
