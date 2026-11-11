# 06: Groupes, ordre, renommage et suppression

**What to build:** Jobierre choisit l'ordre de ses Groupes et celui des Stores dans chaque Groupe, renomme un Groupe et le supprime. L'ordre survit au redémarrage (spec : stories 37, 39, 40, 45 pour ↑/↓, 46, 47, 50 pour le menu ⋯).

**Blocked by:** 05 (Groupes, créer et ranger)

**Status:** ready-for-agent

- [ ] Des boutons ↑/↓ dans le dépliage d'un Store le déplacent dans son Groupe, « Sans groupe » compris.
- [ ] Des boutons ↑/↓ sur l'en-tête d'un Groupe le déplacent parmi les Groupes. « Sans groupe » reste en dernier et ne se déplace pas.
- [ ] Le rangement d'un Store accepte une position (le Store avant lequel l'insérer), pour que le glisser-déposer du ticket 09 réutilise la même action. Le déplacement d'un Groupe accepte de même le Groupe avant lequel l'insérer.
- [ ] Le menu ⋯ d'un Groupe propose Renommer (mêmes règles de nom qu'à la création) et Supprimer, après confirmation. La suppression renvoie ses Stores à la fin de « Sans groupe », dans leur ordre.
- [ ] « Sans groupe » ne se renomme ni ne se supprime.
- [ ] Une action sur un Groupe ou un Store inconnu renvoie à la page avec un message, sans rien modifier. Les actions sont refusées à une identité hors de la liste.
- [ ] Tests par le seam HTTP : ordre des Stores et des Groupes (avec ↑/↓ et avec une position), renommage et ses refus, suppression, cas inconnus, persistance de l'ordre après redémarrage, refus de l'Accès.

## Comments
