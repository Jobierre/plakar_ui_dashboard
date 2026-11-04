# 07: Explorations

**What to build:** Chaque Store a un port d'Exploration stable, pris dans une plage. Un bouton « Explorer » lance le `plakar ui` officiel sur ce Store et donne un lien prêt à l'emploi, avec le jeton inclus. Le service écoute lui-même sur ce port et relaie vers `plakar ui`, ce qui lui permet de vérifier l'identité de l'utilisateur et de mesurer l'inactivité. Une Exploration s'arrête seule après une période d'inactivité. Voir la spec, section « Modules » (Exploration).

**Blocked by:** 03 (État et Seuil de retard), 05 (Accès par identité Tailscale)

**Status:** ready-for-agent

- [ ] La plage de ports vaut 9300-9320 par défaut et peut être réglée par variable d'environnement.
- [ ] Un Store reçoit le premier port libre de la plage. Ce port est persisté dans le fichier d'état et ne change pas tant que le Store existe, y compris après un redémarrage.
- [ ] Le port d'un Store retiré est libéré. Si la plage est épuisée, le Store affiche un message clair à la place du bouton « Explorer ».
- [ ] « Explorer » lance `plakar at @<store> ui -no-spawn` sur un port interne de localhost, distinct du port public. Le service lit le jeton dans la ligne `launching webUI at http://…?plakar_token=<uuid>` (format relevé lors du prototype).
- [ ] Le lien affiché utilise le nom d'hôte public configurable et le port public du Store, et contient le jeton.
- [ ] Le service écoute sur chaque port public en localhost et relaie vers le `plakar ui` correspondant, avec la même vérification d'identité que le tableau de bord.
- [ ] Une Exploration sans trafic pendant la durée d'inactivité (15 min par défaut, configurable) est arrêtée.
- [ ] Le tableau de bord indique les Explorations en cours et permet d'en arrêter une.
- [ ] « Explorer » sur une Exploration déjà en cours redonne son lien, sans lancer un second processus.
- [ ] Le retrait d'un Store arrête son Exploration en cours.
- [ ] Un échec de lancement affiche le message d'erreur de `plakar`.
- [ ] Le jeton n'apparaît jamais dans les journaux du service.
