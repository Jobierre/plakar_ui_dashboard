# 01: Tableau de bord minimal, de bout en bout

**What to build:** Au démarrage, le service liste les Stores de sa config Plakar, fait un Relevé de chacun, puis sert une page avec une ligne par Store montrant le Dernier snapshot complet. Ce ticket pose aussi les fondations que réutilisent tous les suivants : le binaire Go, la configuration par variables d'environnement et le harnais de test (le seam HTTP, le faux `plakar` et l'horloge injectable). Voir la spec, sections « Modules » (Catalogue, Relevé, Web) et « Format d'une ligne `plakar ls -tags` ».

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] Le service démarre, lit la liste des Stores via `plakar store show -json` (voir commentaire) et sert le tableau de bord sur une adresse localhost configurable.
- [x] Le chemin du binaire `plakar` est configurable par variable d'environnement.
- [x] Au démarrage, chaque Store fait l'objet d'un Relevé (`plakar at @<store> ls -tags`), dont le résultat est gardé en mémoire.
- [x] Chaque ligne affiche le nom du Store et, pour le Dernier snapshot : la date en heure locale, l'âge relatif (« il y a 6 h »), l'ID court, la taille, la durée, le chemin source et les tags.
- [x] La lecture d'une ligne `ls` couvre : les tailles avec différentes unités (B, KiB, MiB, GiB…), l'absence de tags, et un Store sans aucun snapshot (ligne vide côté snapshot).
- [x] Un Relevé en échec n'empêche pas l'affichage des autres Stores. La ligne concernée montre le message d'erreur.
- [x] Le harnais de test démarre le service entier, l'interroge en HTTP, remplace `plakar` par un faux binaire qui rejoue des sorties enregistrées et note les arguments reçus, et injecte l'horloge.
- [x] Les tests vérifient le contenu affiché (la bonne valeur dans la bonne ligne), pas la structure interne du code.
- [x] Démo : le service, lancé sur le Mac avec la vraie config Plakar, affiche correctement `@arr`.

## Comments

**2026-11-06, implémentation :**
- `plakar store ls` n'existe pas dans plakar v1.1.6. Jobierre a validé `plakar store show -json` : le Catalogue n'en garde que les noms, triés (l'ordre de plakar est aléatoire). La spec est mise à jour.
- Le service écoute tout de suite, et les Relevés de démarrage tournent en arrière-plan (4 au plus en même temps, 10 min au plus chacun). Une ligne affiche « Relevé en attente » tant que son premier Relevé n'est pas terminé : un Store lent ou bloqué ne retarde ainsi ni la page ni les autres Stores. Le parallélisme et le délai sont des constantes dans `internal/app/app.go`, pas encore des variables d'environnement.
- Un Store sans snapshot affiche « Aucun snapshot » à la place des cellules du Dernier snapshot.
- Le service refuse de démarrer si `PLAKAR_UI_ADDR` n'est pas une adresse localhost, ou si `plakar store show -json` échoue.
- Variables : `PLAKAR_UI_ADDR` (défaut `127.0.0.1:8080`), `PLAKAR_UI_PLAKAR_BIN` (défaut `plakar`).
- Démo du 2026-11-06 sur le Mac avec la vraie config : `@arr` est affiché correctement (`93ebf1a2`, 1.3 GiB, 21s, `/root/docker`, tags `arr`).
