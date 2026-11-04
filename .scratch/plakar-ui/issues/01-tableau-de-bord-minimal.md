# 01: Tableau de bord minimal, de bout en bout

**What to build:** Au démarrage, le service liste les Stores de sa config Plakar, fait un Relevé de chacun, puis sert une page avec une ligne par Store montrant le Dernier snapshot complet. Ce ticket pose aussi les fondations que réutilisent tous les suivants : le binaire Go, la configuration par variables d'environnement et le harnais de test (le seam HTTP, le faux `plakar` et l'horloge injectable). Voir la spec, sections « Modules » (Catalogue, Relevé, Web) et « Format d'une ligne `plakar ls -tags` ».

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Le service démarre, lit la liste des Stores via `plakar store ls` et sert le tableau de bord sur une adresse localhost configurable.
- [ ] Le chemin du binaire `plakar` est configurable par variable d'environnement.
- [ ] Au démarrage, chaque Store fait l'objet d'un Relevé (`plakar at @<store> ls -tags`), dont le résultat est gardé en mémoire.
- [ ] Chaque ligne affiche le nom du Store et, pour le Dernier snapshot : la date en heure locale, l'âge relatif (« il y a 6 h »), l'ID court, la taille, la durée, le chemin source et les tags.
- [ ] La lecture d'une ligne `ls` couvre : les tailles avec différentes unités (B, KiB, MiB, GiB…), l'absence de tags, et un Store sans aucun snapshot (ligne vide côté snapshot).
- [ ] Un Relevé en échec n'empêche pas l'affichage des autres Stores. La ligne concernée montre le message d'erreur.
- [ ] Le harnais de test démarre le service entier, l'interroge en HTTP, remplace `plakar` par un faux binaire qui rejoue des sorties enregistrées et note les arguments reçus, et injecte l'horloge.
- [ ] Les tests vérifient le contenu affiché (la bonne valeur dans la bonne ligne), pas la structure interne du code.
- [ ] Démo : le service, lancé sur le Mac avec la vraie config Plakar, affiche correctement `@arr`.
