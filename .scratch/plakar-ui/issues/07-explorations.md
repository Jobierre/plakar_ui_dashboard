# 07: Explorations

**What to build:** Chaque Store a un port d'Exploration stable, pris dans une plage. Un bouton « Explorer » lance le `plakar ui` officiel sur ce Store et donne un lien prêt à l'emploi, avec le jeton inclus. Le service écoute lui-même sur ce port et relaie vers `plakar ui`, ce qui lui permet de vérifier l'identité de l'utilisateur et de mesurer l'inactivité. Une Exploration s'arrête seule après une période d'inactivité. Voir la spec, section « Modules » (Exploration).

**Blocked by:** 03 (État et Seuil de retard), 05 (Accès par identité Tailscale)

**Status:** resolved

- [x] La plage de ports vaut 9300-9320 par défaut et peut être réglée par variable d'environnement.
- [x] Un Store reçoit le premier port libre de la plage. Ce port est persisté dans le fichier d'état et ne change pas tant que le Store existe, y compris après un redémarrage.
- [x] Le port d'un Store retiré est libéré. Si la plage est épuisée, le Store affiche un message clair à la place du bouton « Explorer ».
- [x] « Explorer » lance `plakar at @<store> ui -no-spawn` sur un port interne de localhost, distinct du port public. Le service lit le jeton dans la ligne `launching webUI at http://…?plakar_token=<uuid>` (format relevé lors du prototype).
- [x] Le lien affiché utilise le nom d'hôte public configurable et le port public du Store, et contient le jeton.
- [x] Le service écoute sur chaque port public en localhost et relaie vers le `plakar ui` correspondant, avec la même vérification d'identité que le tableau de bord.
- [x] Une Exploration sans trafic pendant la durée d'inactivité (15 min par défaut, configurable) est arrêtée.
- [x] Le tableau de bord indique les Explorations en cours et permet d'en arrêter une.
- [x] « Explorer » sur une Exploration déjà en cours redonne son lien, sans lancer un second processus.
- [x] Le retrait d'un Store arrête son Exploration en cours.
- [x] Un échec de lancement affiche le message d'erreur de `plakar`.
- [x] Le jeton n'apparaît jamais dans les journaux du service.

## Comments

**2026-11-07, choix de Jobierre pendant la réalisation :**
- **Port interne** : c'est le port public plus un décalage fixe, `PLAKAR_UI_DECALAGE` (100 par défaut). Il doit dépasser la largeur de la plage et garder les ports internes sous 65536.
  - Vérifié sur plakar v1.1.6 : avec `-addr`, la ligne de lancement recopie l'adresse demandée. Avec `:0`, elle affiche donc `:0`, pas le port réel.
  - Sans `-addr`, plakar choisit un port et l'annonce correctement, mais cette option n'a pas été retenue.
- **« Explorer »** attend que plakar annonce son jeton (3 min au plus), puis affiche le lien, ou le message de plakar en cas d'échec.
- **Inactivité** : une requête en cours (long téléchargement, connexion persistante) compte comme du trafic, et protège l'Exploration tant qu'elle dure.

**2026-11-07, réalisation :**
- Le service écoute sur tous les ports de la plage dès le démarrage. Un port déjà pris empêche le démarrage, avec un message qui nomme `PLAKAR_UI_PORTS`.
- Ports :
  - les ports sont attribués dans l'ordre des Stores et persistés dans le fichier d'état ;
  - après un changement de plage, un port hors de la plage, ou un port en double, est réattribué ;
  - le port d'un Store retiré n'est rendu qu'une fois son Exploration arrêtée.
- Arrêter, ou retirer le Store, coupe aussi une Exploration encore en cours de lancement.
- Arrêter le service arrête toutes les Explorations, par SIGTERM puis par un arrêt forcé au bout de 10 s.
- Toutes les pages du tableau de bord portent `Cache-Control: no-store`, puisqu'elles contiennent les jetons.
- Un test de contrat lance un vrai `plakar ui` à travers le relais : la page répond 200 avec le jeton, et l'API 401 sans lui.
