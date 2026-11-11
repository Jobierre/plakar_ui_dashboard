# Spec : plakar-ui, tableau de bord multi-Stores

Status: ready-for-agent

## Problem Statement

Jobierre possède plusieurs Klosets répartis sur son homelab (des `plakar server` HTTP sur Unraid, un S3 OVH…). Pour savoir si ses backups tournent, il doit aujourd'hui lancer `plakar at @X ls` Store par Store depuis son Mac. Rien ne lui montre d'un coup d'œil quel Store est En retard ou Injoignable, ni si un snapshot a une taille anormale. Pour explorer un Store, il doit lancer `plakar ui` à la main, sur une machine où la config Plakar est présente. La solution officielle multi-Stores (Plakar Control Plane) est commerciale et surdimensionnée pour un homelab.

## Solution

Un service léger, déployé en docker compose sur Unraid et accessible uniquement via Tailscale, qui :

- affiche **une ligne par Store** avec l'**État** (À jour, En retard, Injoignable), le **Dernier snapshot** (date, âge, ID court, taille, durée, chemin source, tags) et l'**Écart** de taille et de durée avec le **Snapshot précédent** ;
- effectue un **Relevé** de chaque Store toutes les heures en arrière-plan, et à la demande ;
- permet d'ajouter, modifier ou retirer un Store en tapant une **Commande de Store** (`plakar store …`) dans l'interface ;
- permet de lancer une **Exploration** (le `plakar ui` officiel) d'un Store sur un port stable, qui s'arrête seule après inactivité ;
- n'autorise que les **Utilisateurs autorisés**, identifiés par leur identité Tailscale.

## User Stories

### Consulter le tableau de bord

1. En tant qu'Utilisateur autorisé, je veux voir une ligne par Store sur une seule page, afin de connaître l'état de toutes mes backups d'un coup d'œil.
2. En tant qu'Utilisateur autorisé, je veux voir l'État de chaque Store (À jour, En retard, Injoignable), afin de repérer immédiatement ce qui demande mon attention.
3. En tant qu'Utilisateur autorisé, je veux voir la date du Dernier snapshot en heure locale, afin de savoir quand la dernière backup a eu lieu.
4. En tant qu'Utilisateur autorisé, je veux voir l'âge du Dernier snapshot (« il y a 6 h »), afin de ne pas avoir à calculer moi-même.
5. En tant qu'Utilisateur autorisé, je veux voir l'ID court du Dernier snapshot, afin de pouvoir le réutiliser dans une commande Plakar.
6. En tant qu'Utilisateur autorisé, je veux voir la taille du Dernier snapshot, afin de vérifier qu'elle est plausible.
7. En tant qu'Utilisateur autorisé, je veux voir la durée de la backup du Dernier snapshot, afin de repérer une backup anormalement lente ou rapide.
8. En tant qu'Utilisateur autorisé, je veux voir le chemin source du Dernier snapshot, afin de savoir ce qui a été sauvegardé.
9. En tant qu'Utilisateur autorisé, je veux voir les tags du Dernier snapshot, afin de distinguer les backups qui partagent un Store.
10. En tant qu'Utilisateur autorisé, je veux voir l'Écart de taille avec le Snapshot précédent (par exemple « −97 % »), afin de détecter une source vide ou un montage absent.
11. En tant qu'Utilisateur autorisé, je veux voir l'Écart de durée avec le Snapshot précédent, afin de détecter un changement de comportement de la backup.
12. En tant qu'Utilisateur autorisé, je veux qu'un Store avec un seul snapshot n'affiche pas d'Écart, afin de ne pas lire une comparaison inventée.
13. En tant qu'Utilisateur autorisé, je veux que l'Écart ne change jamais l'État d'un Store, afin que seul le Seuil de retard décide de « En retard ».
14. En tant qu'Utilisateur autorisé, je veux voir l'heure du dernier Relevé de chaque Store, afin de savoir à quel point l'information est fraîche.
15. En tant qu'Utilisateur autorisé, je veux voir qu'un Relevé est en cours sur un Store, afin de comprendre pourquoi la ligne n'est pas encore à jour (un premier Relevé peut prendre une minute).
16. En tant qu'Utilisateur autorisé, je veux voir le message d'erreur d'un Store Injoignable, afin de diagnostiquer le problème (réseau, passphrase, kloset illisible).
17. En tant qu'Utilisateur autorisé, je veux continuer à voir les dernières valeurs connues d'un Store devenu Injoignable, afin de savoir de quand date la dernière backup réussie connue.

### Relevés

18. En tant qu'Utilisateur autorisé, je veux que chaque Store soit relevé automatiquement toutes les heures, afin que le tableau de bord reste à jour même quand je ne le regarde pas.
19. En tant qu'Utilisateur autorisé, je veux un bouton ↻ par Store qui déclenche un Relevé immédiat, afin de vérifier une backup qui vient de se terminer.
20. En tant qu'Utilisateur autorisé, je veux que deux Relevés d'un même Store ne se chevauchent pas, afin de ne pas surcharger le Kloset ni obtenir des résultats incohérents.
21. En tant qu'Utilisateur autorisé, je veux que les résultats des Relevés survivent au redémarrage du conteneur, afin de ne pas retrouver un tableau de bord vide après un reboot d'Unraid.
22. En tant qu'Utilisateur autorisé, je veux que le cache Plakar survive au redémarrage du conteneur, afin que les Relevés restent rapides (environ 1 s au lieu d'environ 1 min à froid).
23. En tant qu'Utilisateur autorisé, je veux que le Relevé n'ouvre le Store que le temps de la lecture, afin qu'aucun Store ne reste déchiffré en mémoire entre deux Relevés.

### Seuil de retard

24. En tant qu'Utilisateur autorisé, je veux un Seuil de retard par défaut (26 h), afin que mes backups quotidiennes soient surveillées sans configuration.
25. En tant qu'Utilisateur autorisé, je veux modifier le Seuil de retard d'un Store depuis le tableau de bord, afin d'adapter la surveillance d'un Store hebdomadaire (par exemple 8 jours).
26. En tant qu'Utilisateur autorisé, je veux que l'État soit recalculé immédiatement après un changement de Seuil, afin de voir l'effet sans attendre le prochain Relevé.
27. En tant qu'Utilisateur autorisé, je veux qu'un Store passe En retard dès que son Dernier snapshot dépasse le Seuil, même sans nouveau Relevé, afin que le simple passage du temps soit pris en compte.

### Gérer les Stores

28. En tant qu'Utilisateur autorisé, je veux taper une Commande de Store (`plakar store add …`) dans un bloc de l'interface, afin d'ajouter un Store avec la syntaxe officielle de Plakar, qui reste valable même si Plakar évolue.
29. En tant qu'Utilisateur autorisé, je veux voir la sortie et le code de retour de ma Commande de Store, afin de savoir si elle a réussi.
30. En tant qu'Utilisateur autorisé, je veux pouvoir utiliser `add`, `set`, `unset`, `rm` et `show`, afin de gérer complètement mes Stores.
31. En tant qu'Utilisateur autorisé, je veux que `plakar store show -secrets` soit refusé (`show` sans `-secrets` masque les secrets), afin que mes passphrases ne s'affichent jamais dans un navigateur.
32. En tant qu'Utilisateur autorisé, je veux que toute autre commande Plakar (`backup`, `store check`, `store ping`…) ou toute autre commande système soit refusée, afin que le bloc ne devienne pas un terminal distant.
33. En tant qu'Utilisateur autorisé, je veux que les caractères de shell (`;`, `|`, `$(…)`, `&&`…) n'aient aucun effet spécial, afin qu'on ne puisse pas enchaîner une autre commande.
34. En tant qu'Utilisateur autorisé, je veux pouvoir mettre une valeur entre guillemets (une passphrase avec des espaces), afin de saisir n'importe quelle passphrase.
35. En tant qu'Utilisateur autorisé, je veux que mes Commandes de Store ne soient jamais journalisées ni conservées, afin que les passphrases qu'elles contiennent ne traînent nulle part.
36. En tant qu'Utilisateur autorisé, je veux qu'un Store ajouté apparaisse aussitôt dans le tableau de bord, avec un premier Relevé lancé, afin de vérifier tout de suite qu'il fonctionne.
37. En tant qu'Utilisateur autorisé, je veux qu'un Store retiré disparaisse du tableau de bord, afin que la liste reflète toujours la config Plakar. Exception : un Store rangé dans un Groupe y reste comme Store disparu (voir stories 82 à 86).
38. En tant qu'Utilisateur autorisé, je veux que tout Store de la config Plakar du service apparaisse sans déclaration supplémentaire, afin de n'avoir qu'une seule source de vérité.

### Explorations

39. En tant qu'Utilisateur autorisé, je veux un bouton « Explorer » par Store, afin d'ouvrir l'interface officielle de Plakar sur ce Store.
40. En tant qu'Utilisateur autorisé, je veux que chaque Store ait un port d'Exploration attribué automatiquement dans une plage (par défaut 9300-9320), afin de ne rien configurer à la main.
41. En tant qu'Utilisateur autorisé, je veux que le port d'Exploration d'un Store ne change jamais tant que le Store existe, afin que mes favoris restent valables.
42. En tant qu'Utilisateur autorisé, je veux que le port d'un Store retiré soit libéré, afin que la plage ne s'épuise pas.
43. En tant qu'Utilisateur autorisé, je veux un message clair quand la plage de ports est épuisée, afin de savoir qu'il faut l'agrandir.
44. En tant qu'Utilisateur autorisé, je veux accéder à l'Exploration via `https://plakar-ui.<tailnet>.ts.net:<port>` depuis mon PC, afin de l'utiliser comme n'importe quel site.
45. En tant qu'Utilisateur autorisé, je veux que le lien d'Exploration contienne déjà le jeton de `plakar ui`, afin de ne pas avoir à le recopier.
46. En tant qu'Utilisateur autorisé, je veux un nouveau jeton à chaque lancement d'Exploration, afin qu'un ancien lien ne donne plus accès.
47. En tant qu'Utilisateur autorisé, je veux que l'Exploration s'arrête seule après 15 minutes d'inactivité, afin que le Store ne reste pas ouvert indéfiniment.
48. En tant qu'Utilisateur autorisé, je veux voir dans le tableau de bord quelles Explorations sont en cours, et pouvoir en arrêter une, afin de garder le contrôle.
49. En tant qu'Utilisateur autorisé, je veux qu'un clic sur « Explorer » pour une Exploration déjà en cours me redonne son lien, au lieu d'en lancer une seconde, afin d'éviter les doublons.
50. En tant qu'Utilisateur autorisé, je veux que retirer un Store arrête son Exploration en cours, afin qu'aucun processus orphelin ne reste actif.
51. En tant qu'Utilisateur autorisé, je veux voir le message d'erreur si une Exploration ne démarre pas, afin de comprendre pourquoi.

### Accès

52. En tant qu'Utilisateur autorisé, je veux me connecter sans mot de passe grâce à mon identité Tailscale, afin de ne pas gérer un identifiant de plus.
53. En tant que propriétaire du service, je veux définir la liste des Utilisateurs autorisés (emails Tailscale) dans la configuration du compose, afin de contrôler qui accède.
54. En tant que propriétaire du service, je veux qu'une identité Tailscale hors de la liste soit refusée sur toutes les pages, actions et Explorations, afin qu'un autre membre du tailnet n'accède pas à mes backups.
55. En tant que propriétaire du service, je veux qu'aucun port ne soit publié sur le LAN, afin que personne ne contourne Tailscale en forgeant l'en-tête d'identité.
56. En tant que propriétaire du service, je veux que les ports d'Exploration soient protégés par la même vérification d'identité que le tableau de bord, afin que le jeton de `plakar ui` ne soit pas la seule protection.

### Déploiement et exploitation

57. En tant que propriétaire du service, je veux le déployer avec un seul `docker compose up`, afin de l'installer facilement sur Unraid.
58. En tant que propriétaire du service, je veux que la version de Plakar soit figée dans l'image, afin qu'une mise à jour ne casse pas silencieusement la lecture des Relevés.
59. En tant que propriétaire du service, je veux que la config Plakar, le cache Plakar et l'état du service vivent dans des volumes `appdata`, afin de les sauvegarder et de survivre aux mises à jour de l'image.
60. En tant que propriétaire du service, je veux que le conteneur puisse joindre mes `plakar server` par le nom `unraid`, afin de réutiliser les emplacements de Store habituels.
61. En tant que propriétaire du service, je veux régler les valeurs par défaut (Seuil de retard, intervalle de Relevé, inactivité d'Exploration, plage de ports) par variables d'environnement, afin d'adapter le service sans le recompiler.
62. En tant que propriétaire du service, je veux que les journaux du service ne contiennent jamais de passphrase ni de jeton, afin de pouvoir les partager pour un diagnostic.

### Interface (ticket 10)

63. En tant qu'Utilisateur autorisé, je veux un voyant coloré à côté du mot de l'État (🟢 À jour, 🔴 En retard, 🟠 Injoignable), afin de repérer d'un coup d'œil ce qui demande mon attention, sans dépendre de la seule couleur.
64. En tant qu'Utilisateur autorisé, je veux un bandeau en haut de page qui compte les Stores par État, afin de savoir sans faire défiler s'il y a un problème.
65. En tant qu'Utilisateur autorisé, je veux un bouton « Tout relever » dans le bandeau, qui lance un Relevé de chaque Store, sauf ceux dont le Relevé est déjà en cours, afin de tout vérifier après une série de backups.
66. En tant qu'Utilisateur autorisé, je veux que la page se rafraîchisse seule toutes les 5 s tant qu'un Relevé est en cours, et plus du tout ensuite, afin de voir le résultat sans recharger à la main.
67. En tant qu'Utilisateur autorisé, je veux, dans la vue détaillée, une ligne compacte par Store (voyant et État, nom, âge du Dernier snapshot, taille et Écart, heure du dernier Relevé, ↻, Explorer ou lien et Arrêter), avec le reste dans un dépliage (date exacte, ID, durée et Écart, source, tags, Seuil de retard, message de port), afin que la page reste lisible.
68. En tant qu'Utilisateur autorisé, je veux que l'erreur d'un Store Injoignable soit visible sans déplier, et que ses dernières valeurs connues soient atténuées, afin de savoir qu'elles sont anciennes.
69. En tant qu'Utilisateur autorisé, je veux que le dépliage d'un Store s'ouvre seul quand une action sur ce Store vient d'échouer (Seuil, Exploration), afin de voir l'erreur.
70. En tant qu'Utilisateur autorisé, je veux voir l'Écart en texte neutre, signé, avec une flèche ↑/↓ et la valeur du Snapshot précédent en infobulle, sans couleur d'alerte, afin de ne pas le confondre avec l'État.
71. En tant qu'Utilisateur autorisé, je veux que la Commande de Store soit rangée dans une section repliable « Gérer les Stores », ouverte d'elle-même quand un résultat s'affiche, afin qu'elle n'encombre pas le tableau de bord.
72. En tant qu'Utilisateur autorisé, je veux une mise en page pensée pour l'écran du Mac, qui s'empile proprement sur un écran étroit (moins de 700 px), afin de pouvoir aussi consulter depuis un téléphone.
73. En tant qu'Utilisateur autorisé, je veux un thème clair et un thème sombre qui suivent le réglage du système, afin de consulter confortablement le soir.
89. En tant qu'Utilisateur autorisé, je veux une vue simple, en tuiles (bordure à la couleur de l'État, nom, âge du Dernier snapshot en gros, État, taille et Écart, erreur éventuelle), dont un clic ouvre un tiroir latéral avec tous les détails et les actions, afin de faire mon contrôle quotidien d'un coup d'œil.
90. En tant qu'Utilisateur autorisé, je veux basculer entre la vue simple et la vue détaillée depuis le bandeau, avec la vue simple par défaut et mon choix retenu par le navigateur, afin d'avoir le contrôle rapide d'abord et le détail quand j'en ai besoin.

### Groupes (ticket 11)

74. En tant qu'Utilisateur autorisé, je veux ranger mes Stores dans des Groupes nommés, un Groupe au plus par Store, afin d'organiser le tableau de bord à ma façon.
75. En tant qu'Utilisateur autorisé, je veux créer un Groupe (« + Groupe » dans le bandeau), le renommer et le supprimer, avec des noms uniques, afin de gérer mon rangement depuis l'interface.
76. En tant qu'Utilisateur autorisé, je veux que supprimer un Groupe, après confirmation, renvoie ses Stores dans « Sans groupe » et efface ses Stores disparus, sans jamais toucher à la config Plakar, afin que le rangement reste sans risque.
77. En tant qu'Utilisateur autorisé, je veux déplacer un Store d'un Groupe à l'autre et changer son ordre par glisser-déposer, afin de ranger vite.
78. En tant qu'Utilisateur autorisé, je veux réordonner les Groupes en glissant leur en-tête par sa poignée (sections dans la vue détaillée, colonnes dans la vue simple), « Sans groupe » restant toujours en dernier, afin de mettre en haut ce qui compte.
79. En tant qu'Utilisateur autorisé, je veux pouvoir faire tout cela sans JavaScript (liste déroulante « Groupe » et boutons ↑/↓ dans le dépliage), afin que le glisser-déposer reste un confort.
80. En tant qu'Utilisateur autorisé, je veux qu'un Store nouvellement apparu aille en fin de « Sans groupe », et que « Sans groupe » ne s'affiche que s'il contient au moins un Store, afin de repérer ce qui reste à ranger.
81. En tant qu'Utilisateur autorisé, je veux que l'en-tête de chaque Groupe montre le voyant du pire État de ses Stores et ses compteurs, et qu'il se replie (état gardé par le navigateur), afin de condenser ce qui va bien.
82. En tant qu'Utilisateur autorisé, je veux qu'un Store retiré de la config Plakar alors qu'il était dans un Groupe y reste comme Store disparu, grisé, avec « Store supprimé » et un bouton « Retirer du groupe », afin de ne pas perdre mon rangement sur un `rm` suivi d'un `add`.
83. En tant qu'Utilisateur autorisé, je veux qu'un Store qui réapparaît sous le même nom reprenne la place et le Seuil de retard de son Store disparu, afin de n'avoir rien à refaire.
84. En tant qu'Utilisateur autorisé, je veux qu'un Store disparu n'ait ni État, ni Relevé, ni port d'Exploration, ni dernières valeurs affichées, afin de ne pas lire une information périmée ou d'un autre Kloset.
85. En tant qu'Utilisateur autorisé, je veux qu'un Store « Sans groupe » retiré disparaisse simplement, afin que « Sans groupe » ne se remplisse pas de Stores disparus.
86. En tant qu'Utilisateur autorisé, je veux un compteur 👻 des Stores disparus dans le bandeau, hors des compteurs d'État et ignoré par « Tout relever », afin de savoir qu'il reste du ménage à faire.
87. En tant qu'Utilisateur autorisé, je veux cliquer sur le nom d'un Groupe pour ouvrir son focus (`/?groupe=<nom>`, mot-clé réservé pour « Sans groupe »), qui n'affiche que ce Groupe avec un lien « ← Tous les groupes », afin de le mettre en favori.
88. En tant qu'Utilisateur autorisé, je veux qu'en focus tout le bandeau ne porte que sur les Stores du Groupe (compteurs, 👻, et bouton « Relever le groupe »), afin que ce que je vois et ce que je déclenche correspondent.

## Implementation Decisions

### Forme générale
- Un binaire Go unique, qui sert les pages HTML (templates intégrés au binaire) et pilote le binaire `plakar` en sous-processus. Pas de base de données : un petit fichier d'état.
- L'image contient le binaire du service et `plakar` dans une version figée. Pas de rclone.
- Docker compose dans un LXC Docker du homelab (décision du 2026-11-07, au lieu d'Unraid directement), avec deux conteneurs :
  - un sidecar Tailscale en mode TUN noyau (`/dev/net/tun`, `NET_ADMIN`), qui porte le nom `plakar-ui` sur le tailnet et fait le HTTPS avec `tailscale serve` (configuration déclarative, `TS_SERVE_CONFIG`) : 443 vers le tableau de bord, chaque port de la plage vers le même port local. Ce fichier est généré par le service à son démarrage (voir Configuration) ;
  - le service, qui partage le réseau du sidecar et n'écoute que sur localhost.
- Volumes `appdata` : la config Plakar (le `stores.yml` du service, dédié et distinct de celui du Mac), le cache Plakar (indispensable : environ 1 min à froid contre environ 1 s à chaud, plusieurs Go) et le fichier d'état.
- Le nom `unraid` est résolu depuis le conteneur par `extra_hosts` sur le sidecar, vers `UNRAID_IP` (obligatoire : depuis un LXC, `host-gateway` désignerait le LXC et non Unraid).
- Pas de gestionnaire de secrets : les passphrases restent dans la config Plakar standard. Ce choix est assumé.

### Interface (décidée le 2026-11-11)
- Rendu côté serveur par les templates Go, comme avant : mêmes routes `POST` avec redirection vers la page, et mêmes attributs `data-store`, `data-field`, `data-commande`, qui restent le contrat des tests. Si l'interface grossit trop, on passera à une application front séparée sur une API JSON. Garder les routes et les attributs facilite ce passage.
- Une feuille CSS écrite à la main, intégrée au binaire. Aucun framework, aucun CDN, aucune étape de build.
- **JavaScript autorisé là où il simplifie** (décision de Jobierre du 2026-11-11, qui assouplit le « aucun JS obligatoire » initial) : un petit fichier JS fait main, intégré au binaire, sans bibliothèque. Il gère la bascule entre vue simple et vue détaillée et le tiroir de la vue simple, le glisser-déposer des Groupes (API HTML5 native, un `POST` puis rechargement) et les choix retenus par le navigateur (vue, Groupes repliés, dans `localStorage`). Les actions sur le service restent des formulaires `POST`, et le JS n'insère jamais un nom de Store ou de Groupe comme du HTML (`textContent`, jamais `innerHTML`).
- Les deux vues partagent si possible le même balisage, et seule une classe CSS change de mise en page. Chaque Store n'apparaît alors qu'une fois dans la page, et les attributs `data-*` des tests restent uniques. Le tiroir de la vue simple est le dépliage du Store, présenté en panneau latéral.
- Prototype de référence : branche `prototype/ui-variantes`, fichier `internal/web/prototype/ui-variantes.html`, variante D (jamais fusionnée dans main).
- Le rafraîchissement pendant un Relevé en cours passe par `<meta http-equiv="refresh">`, émis seulement dans ce cas.
- Nouvelle route `POST /releves` (Tout relever, ou Relever le groupe en focus).
- Les Groupes, leur ordre, l'ordre des Stores et les Stores disparus sont de l'état propre au service, persisté de façon atomique dans le fichier d'état avec le reste. Un Store disparu garde son Groupe, sa place et son Seuil de retard, mais perd son port et son dernier Relevé.

### Modules
- **Catalogue** : liste des Stores, lue dans la config Plakar via `plakar store show -json` (plakar v1.1.6 n'a pas de `store ls` ; le service ne garde que les noms, les secrets étant de toute façon masqués par plakar), triée par nom (plakar la rend dans un ordre aléatoire), plus l'état propre au service pour chaque Store : Seuil de retard éventuel, port d'Exploration attribué, résultat du dernier Relevé. Il attribue un port au premier Store apparu (le premier libre de la plage) et le libère quand le Store disparaît. Il persiste l'état de façon atomique.
- **Relevé** : pour un Store, lance `plakar at @<store> ls -tags`, garde les deux premières lignes (Plakar trie du plus récent au plus ancien), produit le Dernier snapshot, le Snapshot précédent éventuel et l'Écart, ou une erreur (le Store est alors Injoignable). Un planificateur déclenche un Relevé par Store selon l'intervalle (1 h par défaut), au plus un Relevé à la fois par Store. Il est aussi déclenchable à la demande.
- **État** : une fonction pure (Dernier snapshot, Seuil, maintenant, erreur du dernier Relevé) vers À jour, En retard ou Injoignable. Elle est calculée à l'affichage, avec l'horloge injectée, et n'est donc pas figée au moment du Relevé.
- **Commande de Store** : découpe la ligne saisie en arguments, en gérant les guillemets simples et doubles et sans aucune expansion. Elle accepte uniquement `plakar store` suivi de `add | set | unset | rm | show` (le préfixe `plakar` est facultatif ; `show` est refusé avec l'option `-secrets`, sous toutes ses formes) et exécute le binaire `plakar` directement, sans shell. Elle renvoie la sortie standard, la sortie d'erreur et le code de retour. Elle ne journalise ni ne stocke jamais la commande. Après une réussite, elle rafraîchit le Catalogue et lance un Relevé des nouveaux Stores.
- **Exploration** : lance `plakar at @<store> ui -no-spawn -addr 127.0.0.1:<port interne>`, avec un port interne décalé par rapport au port public pour éviter toute collision. Elle lit le jeton sur la sortie standard. Elle se termine sur inactivité (15 min par défaut), sur un arrêt demandé, ou quand le Store est retiré. Au plus une Exploration par Store.
  - Le service écoute lui-même sur chaque port public de la plage, en localhost, et fait proxy vers le `plakar ui` correspondant. C'est ce qui lui permet de mesurer l'inactivité et d'appliquer la vérification d'identité sur les Explorations.
  - Ligne émise par `plakar ui`, relevée lors du prototype : `launching webUI at http://127.0.0.1:9199?plakar_token=<uuid>`. Le jeton change à chaque lancement, et l'API répond 401 sans lui.
- **Accès** : lit l'en-tête `Tailscale-User-Login` (posé par `tailscale serve`, qui retire les valeurs forgées) et le compare à la liste des Utilisateurs autorisés. Il refuse avec 403 en cas d'absence ou de non-appartenance. Il s'applique au tableau de bord, aux actions et aux ports d'Exploration.
- **Web** : la page du tableau de bord, la mise à jour du Seuil par Store, le bouton ↻, le bloc de Commande de Store, le lancement et l'arrêt d'une Exploration.

### Format d'une ligne `plakar ls -tags`
Relevé lors du prototype, avec plakar v1.1.6 :
```
2026-11-04T17:00:03Z   53f9f62e   1.3 GiB       27s /root/docker tags=arr
```
Les champs sont : date RFC 3339 en UTC, ID court, taille (valeur + unité binaire), durée, chemin source, tags. Les lignes sont triées du plus récent au plus ancien.

### Configuration (variables d'environnement)
- La liste des Utilisateurs autorisés.
- Le Seuil de retard par défaut (26 h).
- L'intervalle de Relevé (1 h).
- L'inactivité d'Exploration (15 min).
- La plage de ports d'Exploration (9300-9320).
- Le décalage entre le port public d'un Store et le port interne de son `plakar ui` (100).
- Le chemin du binaire `plakar`.
- L'emplacement du fichier d'état.
- La clé d'authentification Tailscale, lue par le sidecar. Elle est facultative : la ligne `TS_AUTHKEY` du compose est commentée (choix de Jobierre), le nœud se connecte alors par l'URL affichée dans les journaux du sidecar.
- Le fichier de config `tailscale serve` (`PLAKAR_UI_SERVE_CONFIG`) : le service l'écrit au démarrage, de façon atomique, à partir de son adresse et de sa plage de ports. La plage n'a ainsi qu'une source de vérité, `PLAKAR_UI_PORTS`.
- Côté compose uniquement : `PLAKAR_UI_APPDATA` (dossier des volumes) et `UNRAID_IP`.
- Fixées par l'image : `PLAKAR_UI_ADDR`, `PLAKAR_UI_PLAKAR_BIN`, `PLAKAR_UI_STATE` et `PLAKAR_UI_SERVE_CONFIG`.

## Testing Decisions

- Un bon test vérifie le **comportement observable** du service, pas sa structure interne. Il doit survivre à un refactor des modules.
- **Seam principal : l'interface HTTP du service démarré en entier.** Le test :
  - envoie de vraies requêtes en jouant le rôle de `tailscale serve`, c'est-à-dire qu'il pose lui-même `Tailscale-User-Login` ;
  - remplace `plakar` par un **faux binaire** (désigné par le chemin configurable) qui renvoie des sorties enregistrées sur de vrais Stores (Store normal, Store à un seul snapshot, Store vide, Store Injoignable, `plakar ui` qui émet son jeton) et note les arguments reçus ;
  - **injecte l'horloge** pour tester le Seuil de retard et l'âge.

  Ce seam couvre la lecture des lignes, l'Écart, l'État, l'attribution et la stabilité des ports, la liste blanche et les refus de la Commande de Store, l'absence de journalisation des commandes, le cycle de vie des Explorations et la vérification d'identité.
- **Test de contrat avec le vrai `plakar`**, dans la version figée de l'image. Il crée un Kloset jetable dans un dossier temporaire, fait deux backups d'un petit dossier, puis vérifie que le service relève correctement le Dernier snapshot, le Snapshot précédent et l'Écart. Il échoue si une montée de version de Plakar change le format de sortie.
- Les Groupes, les Stores disparus, le focus et « Tout relever » se testent par le même seam HTTP, en passant par les formulaires sans JS.
- **Non testés automatiquement** : `tailscale serve`, le compose, le JavaScript (bascule de vue, tiroir, glisser-déposer, repli) et le rendu visuel au-delà de « la bonne valeur dans la bonne ligne ». Ils sont vérifiés à la main au déploiement.
- **Prior art** : aucun. Le dépôt ne contient encore aucun code.

## Out of Scope

- Les notifications (ntfy, mail…) : pas en v1.
- Le suivi par source (une ligne par Store × source). Le Dernier snapshot est calculé tous snapshots confondus.
- Les Stores rclone (obsolètes), et rclone dans l'image.
- Les Stores locaux au Mac (`molard`, `sms`, `macbook-images`, `usbpc`).
- Les Stores SFTP : ils seront remplacés par des `plakar server` HTTP, montés par Jobierre hors de ce projet.
- Un gestionnaire de secrets (Vaultwarden ou autre).
- Le déclenchement de backups, checks, syncs ou restaurations depuis le service. Ces actions passent par l'Exploration (le `plakar ui` officiel) ou la ligne de commande.
- `plakar store show -secrets`, `plakar store import`, `plakar store check` et `plakar store ping`.
- L'accès LAN sans Tailscale, et tout autre mode d'authentification.
- L'option « masquer un Store ».
- Les alertes sur l'Écart.

## Further Notes

- **Décisions confirmées par Jobierre après la rédaction :**
  - Un Store **sans aucun snapshot** est affiché **En retard**, puisqu'il n'a pas de Dernier snapshot dans le Seuil.
  - Un Store **Injoignable** garde l'affichage de ses dernières valeurs connues, avec le message d'erreur.
- **Sécurité, risque accepté (2026-11-07) :** les clés passées à `store add` et `store set` ne sont pas filtrées. Or `passphrase_cmd` fait exécuter une commande par plakar à chaque Relevé, dans le conteneur. Un Utilisateur autorisé peut donc exécuter du code dans le conteneur par ce biais. Jobierre l'accepte : voir le ticket 06.
- **Sécurité, mode TUN obligatoire (2026-11-07) :** en mode userspace, tailscaled (v1.104, `wgengine/netstack`, `ProcessLocalIPs`) relaie vers `127.0.0.1:<port>` toute connexion du tailnet sur un port que `tailscale serve` ne gère pas. Un pair du tailnet atteindrait alors le tableau de bord (8080) et les `plakar ui` internes sans passer par serve, en forgeant `Tailscale-User-Login`, ce qui contournerait l'Accès. Le sidecar tourne donc en mode TUN, où localhost reste injoignable depuis le tailnet. Le LXC doit autoriser `/dev/net/tun`. En défense en profondeur, la politique du tailnet n'ouvre vers `plakar-ui` que 443 et la plage d'Exploration, pour les seuls Utilisateurs autorisés (exemple dans `.env.example`).
- **Sécurité :** la config Plakar du service contient les passphrases en clair. Elle doit être lisible seulement par le conteneur (droits `600`), et le volume `appdata` correspondant doit être traité comme sensible.
- **Points vérifiés à l'implémentation (ticket 01, plakar v1.1.6) :**
  - `plakar store ls` n'existe pas (usage : `add|check|import|ping|rm|set|show|unset`). `plakar store show -json` émet une ligne `{"<nom>": {…}}` par Store, secrets remplacés par `********`, dans un ordre aléatoire. Impact sur la Commande de Store, tranché par Jobierre le 2026-11-07 : `ls` est retiré et remplacé par `show` sans `-secrets` (voir le ticket 06).
  - Tailles : `3 B`, `4.9 KiB`, `914 MiB`, `1.3 GiB` (valeur, espace, unité binaire). Durées au format Go (`21s`, `1m11s`, `12m38s`).
  - Sans tag, le champ `tags=` est absent de la ligne ; plusieurs tags s'écrivent `tags=foo,bar`. Le chemin source peut contenir des espaces.
  - Store vide : sortie vide, code 0. Store inconnu : `plakar: could not resolve repository: @<nom>` sur stderr, code 1.
  - Observé une fois : sur une coupure réseau, `ls -tags` a rendu le code 0 avec `plakar: failed to read header … connection reset by peer` sur stdout. La lecture stricte des lignes en fait un Relevé en échec, et non un faux Dernier snapshot.
- **Vérifié au ticket 07 (plakar v1.1.6) :** avec `-addr`, la ligne `launching webUI at …` recopie l'adresse demandée telle quelle. Avec `-addr 127.0.0.1:0`, elle affiche donc `:0`, pas le port réel. Sans `-addr`, plakar choisit un port libre et l'annonce correctement. Jobierre a retenu le décalage fixe (port public + `PLAKAR_UI_DECALAGE`), configurable. SIGTERM arrête `plakar ui` (code 1).
- **Vérifié au ticket 08 (plakar v1.1.6, sous Linux) :**
  - `plakar ui` imprime sa ligne `launching webUI at …` **avant** d'ouvrir son port (`ui/v2/ui.go`). Une Exploration attend donc que le port interne accepte une connexion avant de donner le lien, et le faux plakar des tests reproduit cet ordre.
  - Au premier lancement sur un cache neuf, plakar imprime sur stdout un avis « Welcome to plakar ! » sur son contrôle de mises à jour, qui faussait `store show -json`. Les « cookies » de plakar vivent dans le dossier de cache. Le point d'entrée de l'image lance `plakar -disable-security-check` avant le service (choix de Jobierre : contrôle désactivé, les montées de version se font dans le Dockerfile).
  - plakar se compile statiquement (`CGO_ENABLED=0 go install github.com/PlakarKorp/plakar@v1.1.6`). L'image n'existe que si le test de contrat passe avec ce binaire (étage `contract`).
  - L'image tourne sous l'UID 1000 : les dossiers `appdata` lui appartiennent.
- Les étapes manuelles de déploiement (clé Tailscale, préparation d'`appdata`, `plakar server` qui remplacent les Stores SFTP) pourront être guidées par `/wizard`.
