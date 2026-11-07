# 05: Accès par identité Tailscale

**What to build:** Seuls les Utilisateurs autorisés accèdent au service. Leur identité est lue dans l'en-tête `Tailscale-User-Login`, que pose `tailscale serve` (qui supprime au passage les valeurs forgées), puis comparée à une liste fixée par variable d'environnement. Toute autre requête est refusée.

**Blocked by:** 01 (Tableau de bord minimal, de bout en bout)

**Status:** resolved

- [x] La liste des Utilisateurs autorisés (des emails Tailscale) est lue dans une variable d'environnement.
- [x] Une requête sans en-tête `Tailscale-User-Login` reçoit une réponse 403.
- [x] Une requête dont l'identité n'est pas dans la liste reçoit une réponse 403.
- [x] Le contrôle s'applique à toutes les routes : pages, actions (↻, Seuil) et toute route ajoutée plus tard.
- [x] Protection contre le DNS rebinding : une requête dont l'en-tête `Host` n'est pas attendu reçoit une réponse 403, avant même le contrôle d'identité. Sont acceptés `localhost`, `127.0.0.1`, `[::1]` et le nom Tailscale du service (`plakar-ui.<tailnet>.ts.net`), ce dernier étant lu dans une variable d'environnement. Le port n'est pas pris en compte. Le contrôle s'applique aussi aux ports d'Exploration.
- [x] Le service refuse de démarrer si la liste des Utilisateurs autorisés est vide, avec un message explicite.
- [x] Le service n'écoute que sur localhost. Un test vérifie qu'il ne se lie pas à toutes les interfaces.
- [x] L'identité de l'Utilisateur autorisé est affichée sur le tableau de bord.

## Comments

**2026-11-06, revue de sécurité du ticket 01 :** sans contrôle de `Host`, une page web malveillante ouverte dans un navigateur sur la même machine que le service peut faire pointer son propre nom de domaine vers 127.0.0.1 (DNS rebinding). Elle devient alors « même origine » que le service : elle peut lire ses réponses et poser elle-même l'en-tête `Tailscale-User-Login`. Le contrôle d'identité ne suffit donc pas seul. En production, le risque ne se présente pas, puisqu'aucun navigateur ne tourne dans le réseau du conteneur partagé avec Tailscale. Il existe en dev sur le Mac, et deviendra sérieux avec la Commande de Store (06) et les Explorations (07). À vérifier pendant l'implémentation : la valeur de `Host` que `tailscale serve` transmet réellement au service.

Le critère « Le service n'écoute que sur localhost » est déjà couvert par le ticket 01 (`TestServiceRefusesToListenOutsideLocalhost`).

**2026-11-07, implémenté.**

- Module `internal/acces` : un middleware HTTP générique (`Acces{Users, Hosts}.Handler`) qui enveloppe tout le service dans `app.go`. Toute route y passe, même une route inexistante, qui reçoit 403 et non 404. Le ticket 07 réutilisera la même valeur `Acces` pour envelopper ses proxys d'Exploration. L'identité retenue est lue par `acces.User(ctx)`.
- Ordre des contrôles : le `Host` d'abord, puis la présence de l'identité, puis son appartenance à la liste. Chaque refus est un 403 avec son propre message (« nom d'hôte inattendu », « aucune identité Tailscale », « n'est pas un Utilisateur autorisé »).
- `PLAKAR_UI_UTILISATEURS` : des emails séparés par des virgules, espaces retirés, comparés sans tenir compte de la casse. Une liste vide, ou faite seulement de virgules, empêche le démarrage avec un message qui nomme la variable.
- `PLAKAR_UI_HOTE` : le nom Tailscale du service. Il est accepté en plus de `localhost`, `127.0.0.1` et `[::1]`, quel que soit le port et sans tenir compte de la casse. Le point final d'un nom absolu est retiré, des deux côtés. Une valeur avec un schéma ou un port (`https://…`, `…:443`) empêche le démarrage, car elle ne correspondrait à aucune requête. Si la variable est vide, un avertissement est journalisé au démarrage.
- **Vérifié dans le code source de Tailscale** (`ipn/ipnlocal/serve.go`) : pour un backend TCP, `tailscale serve` conserve le `Host` d'origine (`r.Out.Host = r.In.Host`). Le service reçoit donc `plakar-ui.<tailnet>.ts.net`, et `PLAKAR_UI_HOTE` est **obligatoire en production** (à documenter dans l'exemple d'env du ticket 08). `Tailscale-User-Login` est supprimé, variantes à `_` comprises, puis reposé à partir du WhoIs. Il est absent pour les nœuds taggés et le trafic Funnel, ce qui donne 403.
- Écoute sur localhost : le contrôle du ticket 01 existait déjà. Son test couvre maintenant aussi `:8080`, `[::]:8080` et une adresse du LAN.
- Tableau de bord : « Connecté en tant que <identité> » (`id="utilisateur"`).
- Harnais de test : un Utilisateur autorisé par défaut (`jobierre@example.com`), posé sur chaque requête par `get`/`postForm`. `svc.request` + `svc.do(req, user)` servent aux requêtes brutes. Les deux tests cross-origin envoient maintenant une identité valide, pour que le 403 obtenu vienne bien de l'origine de la requête et non de l'absence d'identité.
- Non retenu : déduire le nom Tailscale dynamiquement (`tailscale status`). Le service n'a pas accès au socket du sidecar. La variable reste explicite.
