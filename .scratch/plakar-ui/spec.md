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
30. En tant qu'Utilisateur autorisé, je veux pouvoir utiliser `add`, `set`, `unset`, `rm` et `ls`, afin de gérer complètement mes Stores.
31. En tant qu'Utilisateur autorisé, je veux que `plakar store show` soit refusé, afin que mes passphrases ne s'affichent jamais dans un navigateur.
32. En tant qu'Utilisateur autorisé, je veux que toute autre commande Plakar (`backup`, `check`…) ou toute autre commande système soit refusée, afin que le bloc ne devienne pas un terminal distant.
33. En tant qu'Utilisateur autorisé, je veux que les caractères de shell (`;`, `|`, `$(…)`, `&&`…) n'aient aucun effet spécial, afin qu'on ne puisse pas enchaîner une autre commande.
34. En tant qu'Utilisateur autorisé, je veux pouvoir mettre une valeur entre guillemets (une passphrase avec des espaces), afin de saisir n'importe quelle passphrase.
35. En tant qu'Utilisateur autorisé, je veux que mes Commandes de Store ne soient jamais journalisées ni conservées, afin que les passphrases qu'elles contiennent ne traînent nulle part.
36. En tant qu'Utilisateur autorisé, je veux qu'un Store ajouté apparaisse aussitôt dans le tableau de bord, avec un premier Relevé lancé, afin de vérifier tout de suite qu'il fonctionne.
37. En tant qu'Utilisateur autorisé, je veux qu'un Store retiré disparaisse du tableau de bord, afin que la liste reflète toujours la config Plakar.
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

## Implementation Decisions

### Forme générale
- Un binaire Go unique, qui sert les pages HTML (templates intégrés au binaire) et pilote le binaire `plakar` en sous-processus. Pas de base de données : un petit fichier d'état.
- L'image contient le binaire du service et `plakar` dans une version figée. Pas de rclone.
- Docker compose sur Unraid, avec deux conteneurs :
  - un sidecar Tailscale, qui porte le nom `plakar-ui` sur le tailnet et fait le HTTPS avec `tailscale serve` (configuration déclarative) : 443 vers le tableau de bord, chaque port de la plage vers le même port local ;
  - le service, qui partage le réseau du sidecar et n'écoute que sur localhost.
- Volumes `appdata` : la config Plakar (le `stores.yml` du service, dédié et distinct de celui du Mac), le cache Plakar (indispensable : environ 1 min à froid contre environ 1 s à chaud, plusieurs Go) et le fichier d'état.
- Le nom `unraid` est résolu depuis le conteneur (par exemple avec `extra_hosts`).
- Pas de gestionnaire de secrets : les passphrases restent dans la config Plakar standard. Ce choix est assumé.

### Modules
- **Catalogue** : liste des Stores, lue dans la config Plakar via `plakar store show -json` (plakar v1.1.6 n'a pas de `store ls` ; le service ne garde que les noms, les secrets étant de toute façon masqués par plakar), triée par nom (plakar la rend dans un ordre aléatoire), plus l'état propre au service pour chaque Store : Seuil de retard éventuel, port d'Exploration attribué, résultat du dernier Relevé. Il attribue un port au premier Store apparu (le premier libre de la plage) et le libère quand le Store disparaît. Il persiste l'état de façon atomique.
- **Relevé** : pour un Store, lance `plakar at @<store> ls -tags`, garde les deux premières lignes (Plakar trie du plus récent au plus ancien), produit le Dernier snapshot, le Snapshot précédent éventuel et l'Écart, ou une erreur (le Store est alors Injoignable). Un planificateur déclenche un Relevé par Store selon l'intervalle (1 h par défaut), au plus un Relevé à la fois par Store. Il est aussi déclenchable à la demande.
- **État** : une fonction pure (Dernier snapshot, Seuil, maintenant, erreur du dernier Relevé) vers À jour, En retard ou Injoignable. Elle est calculée à l'affichage, avec l'horloge injectée, et n'est donc pas figée au moment du Relevé.
- **Commande de Store** : découpe la ligne saisie en arguments, en gérant les guillemets simples et doubles et sans aucune expansion. Elle accepte uniquement `plakar store` suivi de `add | set | unset | rm | ls` (le préfixe `plakar` est facultatif) et exécute le binaire `plakar` directement, sans shell. Elle renvoie la sortie standard, la sortie d'erreur et le code de retour. Elle ne journalise ni ne stocke jamais la commande. Après une réussite, elle rafraîchit le Catalogue et lance un Relevé des nouveaux Stores.
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
- Le chemin du binaire `plakar`.
- L'emplacement du fichier d'état.
- La clé d'authentification Tailscale, lue par le sidecar.

## Testing Decisions

- Un bon test vérifie le **comportement observable** du service, pas sa structure interne. Il doit survivre à un refactor des modules.
- **Seam principal : l'interface HTTP du service démarré en entier.** Le test :
  - envoie de vraies requêtes en jouant le rôle de `tailscale serve`, c'est-à-dire qu'il pose lui-même `Tailscale-User-Login` ;
  - remplace `plakar` par un **faux binaire** (désigné par le chemin configurable) qui renvoie des sorties enregistrées sur de vrais Stores (Store normal, Store à un seul snapshot, Store vide, Store Injoignable, `plakar ui` qui émet son jeton) et note les arguments reçus ;
  - **injecte l'horloge** pour tester le Seuil de retard et l'âge.

  Ce seam couvre la lecture des lignes, l'Écart, l'État, l'attribution et la stabilité des ports, la liste blanche et les refus de la Commande de Store, l'absence de journalisation des commandes, le cycle de vie des Explorations et la vérification d'identité.
- **Test de contrat avec le vrai `plakar`**, dans la version figée de l'image. Il crée un Kloset jetable dans un dossier temporaire, fait deux backups d'un petit dossier, puis vérifie que le service relève correctement le Dernier snapshot, le Snapshot précédent et l'Écart. Il échoue si une montée de version de Plakar change le format de sortie.
- **Non testés automatiquement** : `tailscale serve`, le compose et le rendu visuel au-delà de « la bonne valeur dans la bonne ligne ». Ils sont vérifiés à la main au déploiement.
- **Prior art** : aucun. Le dépôt ne contient encore aucun code.

## Out of Scope

- Les notifications (ntfy, mail…) : pas en v1.
- Le suivi par source (une ligne par Store × source). Le Dernier snapshot est calculé tous snapshots confondus.
- Les Stores rclone (obsolètes), et rclone dans l'image.
- Les Stores locaux au Mac (`molard`, `sms`, `macbook-images`, `usbpc`).
- Les Stores SFTP : ils seront remplacés par des `plakar server` HTTP, montés par Jobierre hors de ce projet.
- Un gestionnaire de secrets (Vaultwarden ou autre).
- Le déclenchement de backups, checks, syncs ou restaurations depuis le service. Ces actions passent par l'Exploration (le `plakar ui` officiel) ou la ligne de commande.
- `plakar store show` et `plakar store import`.
- L'accès LAN sans Tailscale, et tout autre mode d'authentification.
- L'option « masquer un Store ».
- Les alertes sur l'Écart.

## Further Notes

- **Décisions confirmées par Jobierre après la rédaction :**
  - Un Store **sans aucun snapshot** est affiché **En retard**, puisqu'il n'a pas de Dernier snapshot dans le Seuil.
  - Un Store **Injoignable** garde l'affichage de ses dernières valeurs connues, avec le message d'erreur.
- **Sécurité :** la config Plakar du service contient les passphrases en clair. Elle doit être lisible seulement par le conteneur (droits `600`), et le volume `appdata` correspondant doit être traité comme sensible.
- **Points vérifiés à l'implémentation (ticket 01, plakar v1.1.6) :**
  - `plakar store ls` n'existe pas (usage : `add|check|import|ping|rm|set|show|unset`). `plakar store show -json` émet une ligne `{"<nom>": {…}}` par Store, secrets remplacés par `********`, dans un ordre aléatoire. Impact sur la Commande de Store : voir le ticket 06.
  - Tailles : `3 B`, `4.9 KiB`, `914 MiB`, `1.3 GiB` (valeur, espace, unité binaire). Durées au format Go (`21s`, `1m11s`, `12m38s`).
  - Sans tag, le champ `tags=` est absent de la ligne ; plusieurs tags s'écrivent `tags=foo,bar`. Le chemin source peut contenir des espaces.
  - Store vide : sortie vide, code 0. Store inconnu : `plakar: could not resolve repository: @<nom>` sur stderr, code 1.
  - Observé une fois : sur une coupure réseau, `ls -tags` a rendu le code 0 avec `plakar: failed to read header … connection reset by peer` sur stdout. La lecture stricte des lignes en fait un Relevé en échec, et non un faux Dernier snapshot.
- Les étapes manuelles de déploiement (clé Tailscale, préparation d'`appdata`, `plakar server` qui remplacent les Stores SFTP) pourront être guidées par `/wizard`.
