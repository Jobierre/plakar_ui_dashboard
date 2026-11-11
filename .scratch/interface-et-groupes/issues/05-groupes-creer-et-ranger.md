# 05: Groupes, créer et ranger

**What to build:** Jobierre crée des Groupes et y range ses Stores. Le tableau de bord les affiche Groupe par Groupe, « Sans groupe » en dernier, et chaque en-tête de Groupe résume l'État de ses Stores. Le rangement survit au redémarrage (spec : stories 31 à 36, 38, 41, 45 pour le choix du Groupe, 46, 48, 70).

**Blocked by:** 02 (Vue détaillée)

**Status:** ready-for-agent

- [ ] Le Catalogue garde la liste ordonnée des Groupes (nom et membres ordonnés) et celle de « Sans groupe », dans le fichier d'état, de façon atomique et sous le même verrou que le reste.
- [ ] Un fichier d'état de la version actuelle, sans Groupes, se charge sans erreur : aucun Groupe, tous les Stores dans « Sans groupe » par ordre de nom.
- [ ] « + Groupe » dans le bandeau crée un Groupe. Les noms sont nettoyés des espaces de bord, non vides, uniques et différents du mot réservé à « Sans groupe » ; un nom refusé renvoie à la page avec un message, sans rien modifier.
- [ ] Une liste déroulante « Groupe » dans le dépliage d'un Store le range dans un Groupe ou dans « Sans groupe », à la fin. Un Store appartient à un Groupe au plus.
- [ ] Un Store nouvellement apparu dans la config va à la fin de « Sans groupe ».
- [ ] « Sans groupe » est affiché en dernier, et seulement s'il contient au moins un Store.
- [ ] L'en-tête de chaque Groupe montre le voyant du pire État de ses Stores (En retard, puis Injoignable, puis À jour) et ses compteurs.
- [ ] Rien de tout cela ne touche à la config Plakar. Les actions sont refusées à une identité hors de la liste.
- [ ] Tests par le seam HTTP : création et refus de noms, rangement, Store nouveau en fin de « Sans groupe », en-tête de Groupe, persistance après redémarrage, chargement d'un fichier d'état sans Groupes, refus de l'Accès.

## Comments
