# Spec : interface du tableau de bord et Groupes de Stores

Status: ready-for-agent

Fonctionnalité qui s'appuie sur le service décrit dans `.scratch/plakar-ui/spec.md` (tickets 01 à 09, tous résolus). Vocabulaire : `GLOSSARY.md` (État, Relevé, Écart, Seuil de retard, Groupe, Store disparu…). Conçue par grilling le 2026-11-11 (questions Q1 à Q35), puis validée sur un prototype.

## Problem Statement

Le tableau de bord de plakar-ui fonctionne, mais c'est du HTML brut : un tableau de 21 colonnes, sans mise en forme. Pour savoir si tout va bien, Jobierre doit lire chaque ligne et repérer le mot « En retard » ou « Injoignable » au milieu des dates, des tailles et des formulaires. Pour vérifier ses backups après une série de sauvegardes, il doit cliquer ↻ Store par Store. Après un Relevé, la page ne bouge pas tant qu'il ne la recharge pas. La Commande de Store, rarement utilisée, occupe une place permanente. Enfin, les Stores s'affichent par ordre alphabétique, sans moyen de ranger ensemble ceux qui vont ensemble (ceux d'Unraid, ceux hors site…). Plus il ajoute de Stores, plus le tableau de bord devient difficile à lire.

## Solution

Le tableau de bord devient une interface lisible d'un coup d'œil, pensée d'abord pour l'écran du Mac :

- chaque Store porte un **voyant** de couleur à côté du mot de son État : 🟢 À jour, 🔴 En retard, 🟠 Injoignable ;
- un **bandeau** en haut de page compte les Stores par État et propose **Tout relever** ;
- deux **vues** au choix : une **vue simple** en tuiles, affichée par défaut, pour le contrôle quotidien, et une **vue détaillée** en liste dense ;
- les Stores se rangent dans des **Groupes** nommés, dans l'ordre que Jobierre veut, par glisser-déposer ;
- un Groupe peut s'ouvrir en **focus**, avec une adresse qu'on peut mettre en favori ;
- un Store retiré de la config Plakar alors qu'il était rangé reste à sa place comme **Store disparu**, pour qu'un `store rm` suivi d'un `store add` ne défasse pas le rangement.

Rien ne change sur ce que le service mesure. L'État, le Relevé, l'Écart et le Seuil de retard gardent leurs définitions, et l'Écart ne déclenche toujours aucune alerte.

## User Stories

### Voir l'état d'un coup d'œil

1. En tant qu'Utilisateur autorisé, je veux un voyant coloré à côté du mot de l'État de chaque Store (🟢 À jour, 🔴 En retard, 🟠 Injoignable), afin de repérer immédiatement ce qui demande mon attention.
2. En tant qu'Utilisateur autorisé, je veux que l'État soit toujours écrit en toutes lettres à côté du voyant, afin de ne pas dépendre de la seule couleur (daltonisme, lecteur d'écran, capture en noir et blanc).
3. En tant qu'Utilisateur autorisé, je veux un bandeau en haut de page qui compte les Stores À jour, En retard et Injoignables, afin de savoir sans faire défiler s'il y a un problème.
4. En tant qu'Utilisateur autorisé, je veux que le bandeau reste visible quand je fais défiler la page, afin de garder le résumé et le bouton Tout relever sous la main.
5. En tant qu'Utilisateur autorisé, je veux voir dans le bandeau sous quelle identité je suis connecté, afin de vérifier que l'Accès m'a bien reconnu.
6. En tant qu'Utilisateur autorisé, je veux qu'un Store dont le premier Relevé n'est pas terminé apparaisse comme « Premier Relevé » et non comme En retard, afin de ne pas prendre un Store tout juste ajouté pour un problème.
7. En tant qu'Utilisateur autorisé, je veux que l'erreur d'un Store Injoignable soit visible sans rien déplier, afin de savoir tout de suite pourquoi il est injoignable.
8. En tant qu'Utilisateur autorisé, je veux que les dernières valeurs connues d'un Store Injoignable soient atténuées, afin de comprendre qu'elles sont anciennes.
9. En tant qu'Utilisateur autorisé, je veux voir l'Écart en texte neutre, signé (« −97 % », « +2 % »), avec une flèche ↑, ↓ ou →, afin de lire sa direction d'un coup d'œil.
10. En tant qu'Utilisateur autorisé, je veux voir la valeur du Snapshot précédent en infobulle sur l'Écart, afin de comparer sans déplier.
11. En tant qu'Utilisateur autorisé, je veux que l'Écart n'ait jamais de couleur d'alerte, afin de ne pas le confondre avec l'État, seul juge de « En retard ».
12. En tant qu'Utilisateur autorisé, je veux un thème clair et un thème sombre qui suivent automatiquement le réglage de mon système, afin de consulter confortablement le soir.

### Deux vues

13. En tant qu'Utilisateur autorisé, je veux une vue simple où chaque Store est une tuile (bordure à la couleur de l'État, nom, âge du Dernier snapshot en gros, État, taille et Écart, erreur éventuelle), afin de faire mon contrôle quotidien en quelques secondes.
14. En tant qu'Utilisateur autorisé, je veux qu'un clic sur une tuile ouvre un tiroir latéral avec tous les détails et toutes les actions du Store, afin de creuser sans quitter la vue simple.
15. En tant qu'Utilisateur autorisé, je veux une vue détaillée où chaque Store est une ligne compacte (voyant et État, nom, âge du Dernier snapshot, taille et Écart de taille, heure du dernier Relevé ou « Relevé en cours », ↻, Explorer ou lien d'Exploration et Arrêter), afin de comparer les Stores en colonnes alignées.
16. En tant qu'Utilisateur autorisé, je veux déplier une ligne de la vue détaillée pour voir la date exacte, l'ID court, la durée et son Écart, le chemin source, les tags, le Seuil de retard avec son formulaire et le message de port, afin d'avoir tout le détail sans encombrer la liste.
17. En tant qu'Utilisateur autorisé, je veux basculer entre la vue simple et la vue détaillée par un sélecteur « Simple | Détaillée » dans le bandeau, afin de passer du coup d'œil au détail en un clic.
18. En tant qu'Utilisateur autorisé, je veux que la vue simple s'affiche par défaut, afin que mon contrôle quotidien soit immédiat.
19. En tant qu'Utilisateur autorisé, je veux que mon choix de vue soit retenu par mon navigateur, afin de garder la vue détaillée sur le Mac et la vue simple sur le téléphone si je le souhaite.
20. En tant qu'Utilisateur autorisé, je veux que le dépliage (ou le tiroir) d'un Store s'ouvre de lui-même quand une action sur ce Store vient d'échouer (Seuil invalide, Exploration qui ne démarre pas), afin de voir l'erreur sans la chercher.
21. En tant qu'Utilisateur autorisé, je veux une mise en page pensée pour l'écran du Mac, qui s'empile proprement sous 700 px de large, afin de pouvoir quand même consulter depuis un téléphone.

### Relever

22. En tant qu'Utilisateur autorisé, je veux un bouton « Tout relever » dans le bandeau, qui lance un Relevé de chaque Store, afin de tout vérifier après une série de backups.
23. En tant qu'Utilisateur autorisé, je veux que Tout relever ignore les Stores dont un Relevé est déjà en cours, afin que deux Relevés d'un même Store ne se chevauchent jamais.
24. En tant qu'Utilisateur autorisé, je veux que Tout relever soit désactivé quand tous les Stores affichés ont déjà un Relevé en cours, afin de ne pas cliquer dans le vide.
25. En tant qu'Utilisateur autorisé, je veux garder le bouton ↻ par Store, afin de vérifier une seule backup qui vient de finir.
26. En tant qu'Utilisateur autorisé, je veux que la page se rafraîchisse seule toutes les 5 s tant qu'au moins un Relevé est en cours, afin de voir les résultats arriver sans recharger.
27. En tant qu'Utilisateur autorisé, je veux que ce rafraîchissement s'arrête dès qu'aucun Relevé n'est en cours, afin que la page ne clignote pas pour rien et ne m'interrompe pas pendant une saisie.

### Gérer les Stores

28. En tant qu'Utilisateur autorisé, je veux que la Commande de Store soit rangée dans une section repliable « Gérer les Stores », fermée par défaut, afin qu'elle n'encombre pas le tableau de bord.
29. En tant qu'Utilisateur autorisé, je veux que cette section s'ouvre d'elle-même quand un résultat de Commande de Store s'affiche (refus, erreur ou sortie), afin de voir ce que ma commande a donné.
30. En tant qu'Utilisateur autorisé, je veux voir la sortie standard et la sortie d'erreur dans des blocs façon terminal, afin de les lire comme dans mon shell.

### Ranger les Stores en Groupes

31. En tant qu'Utilisateur autorisé, je veux ranger mes Stores dans des Groupes nommés, afin d'organiser le tableau de bord à ma façon (par machine, par lieu, par usage…).
32. En tant qu'Utilisateur autorisé, je veux qu'un Store appartienne à un Groupe au plus, afin que chaque Store n'apparaisse qu'une fois.
33. En tant qu'Utilisateur autorisé, je veux que les Stores rangés nulle part apparaissent sous « Sans groupe », toujours en dernier, afin de voir ce qu'il me reste à ranger.
34. En tant qu'Utilisateur autorisé, je veux que « Sans groupe » ne s'affiche que s'il contient au moins un Store, afin de ne pas voir une section vide une fois tout rangé.
35. En tant qu'Utilisateur autorisé, je veux qu'un Store nouvellement apparu dans la config Plakar se place à la fin de « Sans groupe », afin de le repérer facilement.
36. En tant qu'Utilisateur autorisé, je veux créer un Groupe depuis un bouton « + Groupe » du bandeau, afin de commencer à ranger.
37. En tant qu'Utilisateur autorisé, je veux renommer un Groupe depuis son menu ⋯, afin de corriger un nom sans tout refaire.
38. En tant qu'Utilisateur autorisé, je veux que les noms de Groupe soient uniques, non vides, et différents du mot réservé à « Sans groupe », afin qu'aucun focus ne soit ambigu.
39. En tant qu'Utilisateur autorisé, je veux supprimer un Groupe depuis son menu ⋯, après confirmation, afin de défaire un rangement qui ne me sert plus.
40. En tant qu'Utilisateur autorisé, je veux que supprimer un Groupe renvoie ses Stores à la fin de « Sans groupe », dans leur ordre, et efface ses Stores disparus, afin de ne rien perdre d'autre que le rangement.
41. En tant qu'Utilisateur autorisé, je veux que créer, renommer, supprimer un Groupe ou y ranger un Store ne touche jamais à la config Plakar, afin que le rangement soit sans risque pour mes backups.
42. En tant qu'Utilisateur autorisé, je veux glisser-déposer un Store dans un autre Groupe ou à une autre place de son Groupe, afin de ranger vite.
43. En tant qu'Utilisateur autorisé, je veux glisser-déposer un Groupe par sa poignée ⠿ pour changer l'ordre des Groupes, afin de mettre en haut ce qui compte.
44. En tant qu'Utilisateur autorisé, je veux que le glisser-déposer marche dans les deux vues (sections dans la vue détaillée, colonnes de tuiles dans la vue simple), afin de ranger depuis la vue où je suis.
45. En tant qu'Utilisateur autorisé, je veux aussi pouvoir ranger sans glisser (liste déroulante « Groupe » et boutons ↑/↓ dans le dépliage d'un Store, et ↑/↓ sur un Groupe), afin d'avoir une voie précise et toujours disponible.
46. En tant qu'Utilisateur autorisé, je veux que l'ordre des Groupes et l'ordre des Stores dans chaque Groupe, « Sans groupe » compris, survivent au redémarrage du conteneur, afin de ne pas tout ranger à nouveau après un reboot.
47. En tant qu'Utilisateur autorisé, je veux que « Sans groupe » ne puisse être ni renommé, ni supprimé, ni déplacé, afin qu'il reste un repère fixe.
48. En tant qu'Utilisateur autorisé, je veux que l'en-tête de chaque Groupe montre le voyant du pire État de ses Stores (En retard, puis Injoignable, puis À jour) et ses compteurs, afin de savoir quel Groupe ouvrir.
49. En tant qu'Utilisateur autorisé, je veux replier un Groupe par son chevron ▾, et que mon navigateur s'en souvienne, afin de condenser ce qui va bien.
50. En tant qu'Utilisateur autorisé, je veux que les quatre gestes de l'en-tête d'un Groupe soient distincts (poignée pour glisser, nom pour le focus, chevron pour replier, menu ⋯ pour renommer ou supprimer), afin de ne jamais déclencher l'un en voulant l'autre.

### Focus sur un Groupe

51. En tant qu'Utilisateur autorisé, je veux cliquer sur le nom d'un Groupe pour n'afficher que ce Groupe, afin de me concentrer sur une partie de mon infra.
52. En tant qu'Utilisateur autorisé, je veux que le focus ait sa propre adresse, afin de la mettre en favori et que le bouton Précédent du navigateur fonctionne.
53. En tant qu'Utilisateur autorisé, je veux un lien « ← Tous les groupes » dans le focus, afin de revenir à la vue d'ensemble.
54. En tant qu'Utilisateur autorisé, je veux pouvoir aussi ouvrir le focus de « Sans groupe », afin de retrouver d'un coup d'œil les Stores à ranger.
55. En tant qu'Utilisateur autorisé, je veux qu'en focus tout le bandeau ne porte que sur les Stores du Groupe (compteurs d'État, compteur 👻 et bouton), afin que ce que je vois et ce que je déclenche correspondent.
56. En tant qu'Utilisateur autorisé, je veux qu'en focus le bouton du bandeau s'appelle « Relever le groupe » et ne relève que les Stores du Groupe, afin de savoir exactement ce qu'il va faire.
57. En tant qu'Utilisateur autorisé, je veux que le focus d'un Groupe qui n'existe pas (renommé, supprimé, adresse fausse) me le dise et me propose de revenir à tous les Groupes, afin qu'un vieux favori ne donne pas une page vide sans explication.

### Stores disparus

58. En tant qu'Utilisateur autorisé, je veux qu'un Store retiré de la config Plakar alors qu'il était rangé dans un Groupe reste à sa place dans ce Groupe comme Store disparu, afin de ne pas perdre mon rangement sur un `store rm` suivi d'un `store add`.
59. En tant qu'Utilisateur autorisé, je veux qu'un Store disparu soit grisé, porte la mention « Store supprimé » et un bouton « Retirer du groupe », afin de le reconnaître et de faire le ménage quand je le décide.
60. En tant qu'Utilisateur autorisé, je veux qu'un Store qui réapparaît dans la config Plakar sous le même nom reprenne la place et le Seuil de retard de son Store disparu, afin de n'avoir rien à refaire.
61. En tant qu'Utilisateur autorisé, je veux qu'un Store qui réapparaît obtienne un port d'Exploration et un premier Relevé comme tout Store ajouté, afin de vérifier tout de suite qu'il fonctionne.
62. En tant qu'Utilisateur autorisé, je veux qu'un Store disparu n'ait ni État, ni Relevé, ni port d'Exploration, ni dernières valeurs affichées, afin de ne pas lire une information périmée, peut-être d'un autre Kloset.
63. En tant qu'Utilisateur autorisé, je veux qu'un Store « Sans groupe » retiré de la config disparaisse simplement, afin que « Sans groupe » ne se remplisse pas de Stores disparus.
64. En tant qu'Utilisateur autorisé, je veux un compteur 👻 des Stores disparus dans le bandeau, à part des compteurs d'État, afin de savoir qu'il reste du ménage à faire.
65. En tant qu'Utilisateur autorisé, je veux que Tout relever et Relever le groupe ignorent les Stores disparus, afin de ne pas déclencher un Relevé qui échouerait forcément.
66. En tant qu'Utilisateur autorisé, je veux que les Stores disparus survivent au redémarrage du conteneur, afin que le rangement tienne même si je redémarre entre le `rm` et le `add`.

### Sécurité et robustesse

67. En tant que propriétaire du service, je veux que toutes les nouvelles pages et actions (Tout relever, Groupes, focus, feuille de style, script) passent par la même vérification d'identité que le reste, afin qu'un membre du tailnet hors de la liste n'y accède pas.
68. En tant que propriétaire du service, je veux que le script de l'interface n'insère jamais un nom de Store ou de Groupe comme du HTML, afin qu'un nom piégé ne puisse pas exécuter de code dans mon navigateur.
69. En tant que propriétaire du service, je veux que la feuille de style et le script soient intégrés au binaire et servis sans CDN, afin que l'interface marche même si le conteneur n'a pas accès à Internet.
70. En tant qu'Utilisateur autorisé, je veux que toutes les actions de rangement soient atomiques dans le fichier d'état, afin qu'un arrêt brutal ne laisse jamais un rangement à moitié écrit.

## Implementation Decisions

### Forme générale
- Le rendu reste côté serveur, par les templates Go du module Web. Les routes existantes et les attributs `data-store`, `data-field` et `data-commande` sont gardés : ils sont le contrat des tests. Si l'interface grossit trop, on passera plus tard à une application front séparée sur une API JSON (option écartée pour l'instant), et garder ces routes facilitera ce passage.
- Une feuille CSS et un petit fichier JavaScript, écrits à la main, sans bibliothèque ni étape de build, sont intégrés au binaire et servis par le service lui-même, derrière l'Accès.
- **Le JavaScript est autorisé là où il simplifie** (décision de Jobierre, qui assouplit le « aucun JS obligatoire » du grilling). Il gère la bascule de vue, le tiroir de la vue simple, le glisser-déposer et ce que le navigateur retient (vue choisie, Groupes repliés, dans `localStorage`). Toute action sur le service reste un formulaire `POST` suivi d'une redirection vers la page. Le glisser-déposer envoie le même `POST` que le formulaire de repli, puis recharge la page. Le script manipule les noms par `textContent`, jamais par `innerHTML`.
- **Les deux vues partagent le même balisage** : une classe sur la page choisit la mise en page (tuiles ou liste). Chaque Store n'apparaît donc qu'une fois dans le HTML, et ses attributs `data-*` restent uniques. Le tiroir de la vue simple est le dépliage du Store, présenté en panneau latéral. Sans JavaScript, la page s'affiche dans la vue par défaut (simple), et tout reste utilisable par les formulaires.
- Thème clair ou sombre par `prefers-color-scheme`, sans bouton de bascule.
- Le rafraîchissement automatique passe par une balise `<meta http-equiv="refresh" content="5">`, émise par le serveur seulement quand au moins un Store de la vue a un Relevé en cours.

### Module Web
- La page reçoit, en plus des lignes, les Groupes dans leur ordre avec leurs Stores (et Stores disparus) dans leur ordre, « Sans groupe » en dernier, les compteurs du bandeau calculés sur les Stores de la vue, le pire État de chaque Groupe et le drapeau « au moins un Relevé en cours ».
- Le focus est un paramètre de la page d'accueil : `/?groupe=<nom>`. Un mot réservé désigne « Sans groupe » (indicatif : `_sans`). Un focus inconnu affiche un message et le lien « ← Tous les groupes ».
- Le dépliage s'ouvre d'office pour un Store dont l'action vient d'échouer, et la section « Gérer les Stores » pour un résultat de Commande de Store, comme les erreurs s'affichent aujourd'hui.
- Contrat des nouvelles actions, toutes en `POST` avec redirection (noms de route indicatifs, à fixer au ticket) :
  - relever plusieurs Stores : sans paramètre, tous les Stores ; avec un Groupe (ou le mot réservé), les Stores de ce Groupe ;
  - créer un Groupe (nom) ;
  - renommer un Groupe (nouveau nom) ;
  - supprimer un Groupe ;
  - déplacer un Groupe (vers le haut, vers le bas, ou avant un autre Groupe) ;
  - ranger un Store (Groupe cible ou « Sans groupe », et éventuellement le Store avant lequel l'insérer ; sans position, à la fin) ;
  - monter ou descendre un Store dans son Groupe ;
  - retirer un Store disparu de son Groupe.
- Une action sur un Groupe ou un Store inconnu, ou un nom de Groupe refusé (vide, en double, réservé), renvoie à la page avec un message d'erreur, sans rien modifier.

### Module Relevé (planificateur)
- Nouvelle opération « relever ces Stores », qui déclenche un Relevé à la demande pour chaque nom donné, en sautant ceux dont un Relevé est déjà en cours. Elle réutilise le déclenchement par Store existant, avec la même garantie de non-chevauchement.

### Module Catalogue (état du service)
- Le Catalogue devient le gardien du rangement : la liste ordonnée des Groupes, chacun avec son nom et la liste ordonnée de ses membres, et la liste ordonnée de « Sans groupe ».
- Un membre de Groupe absent de la config Plakar est un Store disparu. Le Catalogue garde alors son Seuil de retard, libère son port d'Exploration, oublie son dernier Relevé et ne le propose plus au planificateur.
- **Changement de comportement** : aujourd'hui, au chargement, l'état d'un Store absent de la config est ignoré puis effacé à l'enregistrement suivant. Désormais, le Seuil d'un Store disparu est conservé tant qu'il est membre d'un Groupe. Un Store « Sans groupe » absent de la config est oublié comme avant.
- À chaque rafraîchissement du Catalogue (démarrage, Commande de Store réussie) :
  - un nom de la config qui est membre d'un Groupe redevient un Store à sa place, avec son Seuil, un nouveau port et un premier Relevé ;
  - un nom nouveau va à la fin de « Sans groupe » ;
  - un Store disparu de la config devient Store disparu s'il est dans un Groupe, et sinon il est oublié.
- Supprimer un Groupe envoie ses Stores à la fin de « Sans groupe », dans leur ordre, et oublie ses Stores disparus, Seuil compris.
- Schéma du fichier d'état : il garde son objet `stores` (Seuil, Relevé et port par nom) et gagne deux champs, la liste ordonnée des Groupes (nom et liste ordonnée des membres) et la liste ordonnée de « Sans groupe ». Un fichier d'état sans ces champs (version actuelle) se charge sans erreur : aucun Groupe, et tous les Stores dans « Sans groupe », par ordre de nom.
- Les noms de Groupe sont nettoyés des espaces de début et de fin, non vides, uniques, et différents du mot réservé.
- Toute modification du rangement est enregistrée de façon atomique, sous le même verrou que le reste de l'état du Catalogue.

### Ce que le prototype a fixé
- Le prototype (branche `prototype/ui-variantes`, variante D) a servi à choisir les vues. La vue détaillée vient de la variante A (liste dense, Groupes en sections, dépliage par ligne), la vue simple de la variante B (Groupes en colonnes de tuiles, tiroir latéral). La variante C (arborescence à gauche et fiche à droite) a été écartée.
- Il sert de référence visuelle (couleurs, densité, gestes de l'en-tête de Groupe), pas de code à reprendre : il a été écrit sans tests ni échappement systématique.

## Testing Decisions

- Un bon test vérifie le **comportement observable** du service, pas sa structure : ce que montre la page et ce que changent les actions. Il doit survivre à une refonte des templates ou du Catalogue.
- **Un seul seam : l'interface HTTP du service démarré en entier**, celui des tests existants. Le harnais démarre le service avec un faux `plakar` et une horloge injectée, joue le rôle de `tailscale serve` en posant l'identité, envoie des `GET` et des `POST` de formulaire, et lit la page par ses attributs `data-*`. Aucun nouveau seam n'est créé : le rangement se teste par les formulaires de repli, que le glisser-déposer réutilise.
- **Prior art** : les tests de la page d'accueil (une ligne par Store, valeurs par `data-field`), ceux des Relevés à la demande (non-chevauchement, Relevé en cours), ceux de la persistance (redémarrage du service sur le même fichier d'état, fichier d'état illisible), ceux de la Commande de Store (le faux `plakar` change sa liste de Stores, puis le Catalogue se rafraîchit) et ceux de l'Accès (identité hors liste refusée).
- Comportements à couvrir :
  - les compteurs du bandeau, sur tous les Stores puis en focus ;
  - Tout relever : chaque Store relevé, Stores déjà en cours sautés, Stores disparus ignorés ; Relever le groupe en focus ;
  - le rafraîchissement présent pendant un Relevé en cours, absent sinon ;
  - chaque Store présent une seule fois dans la page, dans les deux vues ;
  - création, renommage, suppression d'un Groupe, avec les refus (vide, en double, réservé, inconnu) ;
  - rangement d'un Store, ordre des Stores et des Groupes, nouveau Store en fin de « Sans groupe » ;
  - le cycle de vie d'un Store disparu : `rm` alors qu'il est rangé, affichage sans État ni valeurs, port libéré, `add` du même nom qui reprend sa place et son Seuil, « Retirer du groupe » ; et un Store « Sans groupe » retiré qui disparaît ;
  - la persistance du rangement et des Stores disparus après un redémarrage ;
  - le chargement d'un fichier d'état de la version actuelle, sans Groupes ;
  - le focus d'un Groupe, celui de « Sans groupe » et celui d'un Groupe inconnu ;
  - l'Accès refusé sur chaque nouvelle action et sur la feuille de style et le script.
- **Non testés automatiquement** : le rendu visuel, la bascule de vue, le tiroir, le glisser-déposer et le repli des Groupes, c'est-à-dire tout ce que fait le JavaScript. Ils sont vérifiés à la main dans le navigateur, en comparant au prototype.

## Out of Scope

- Une application front séparée (React, Vue…) et une API JSON : à envisager seulement si l'interface devient trop grosse.
- Un Store dans plusieurs Groupes, ou des Groupes imbriqués.
- Un Seuil de retard, un intervalle de Relevé ou des droits d'accès par Groupe : un Groupe n'a aucun effet sur l'État, le Relevé ou le Seuil.
- Des alertes ou des couleurs sur l'Écart.
- Le déclenchement de syncs (`plakar sync`), de backups, de checks ou de restaurations depuis le service.
- Les notifications.
- L'option « masquer un Store ».
- Garder côté service la vue choisie ou l'état replié des Groupes : c'est l'affaire du navigateur.
- Un bouton de bascule clair/sombre.
- Le glisser-déposer au doigt sur téléphone : l'API HTML5 native ne le gère pas, et les formulaires de repli suffisent.
- Le tri automatique des Stores (par État, par âge…) : l'ordre est celui que Jobierre a choisi.

## Further Notes

- **Vocabulaire fixé pendant la conception** (voir `GLOSSARY.md`) : on dit « Relevé » et non « sync », « État » et non « décalage », « Store disparu » et non « Store fantôme ».
- **Tickets** : neuf tranches verticales dans `issues/`, de `01` (Feuille de style, voyant et bandeau) à `09` (Glisser-déposer et repli des Groupes). Les tickets 01 à 04 ne connaissent pas encore les Groupes : la page reste une seule liste, dans l'ordre des noms.
- Cette spec remplace la section « Interface » et les stories 63 à 90 ajoutées le 2026-11-11 à `.scratch/plakar-ui/spec.md`, qui renvoie maintenant ici.
- **Sécurité, rappel du prototype** : une revue automatique a relevé une injection possible dans le prototype (un nom de Groupe lu dans l'adresse et inséré comme du HTML). Les templates Go échappent déjà tout. Le seul risque restant est le script, d'où la story 68 et la règle `textContent`.
