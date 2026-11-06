# 05: Accès par identité Tailscale

**What to build:** Seuls les Utilisateurs autorisés accèdent au service. Leur identité est lue dans l'en-tête `Tailscale-User-Login`, que pose `tailscale serve` (qui supprime au passage les valeurs forgées), puis comparée à une liste fixée par variable d'environnement. Toute autre requête est refusée.

**Blocked by:** 01 (Tableau de bord minimal, de bout en bout)

**Status:** ready-for-agent

- [ ] La liste des Utilisateurs autorisés (des emails Tailscale) est lue dans une variable d'environnement.
- [ ] Une requête sans en-tête `Tailscale-User-Login` reçoit une réponse 403.
- [ ] Une requête dont l'identité n'est pas dans la liste reçoit une réponse 403.
- [ ] Le contrôle s'applique à toutes les routes : pages, actions (↻, Seuil) et toute route ajoutée plus tard.
- [ ] Protection contre le DNS rebinding : une requête dont l'en-tête `Host` n'est pas attendu reçoit une réponse 403, avant même le contrôle d'identité. Sont acceptés `localhost`, `127.0.0.1`, `[::1]` et le nom Tailscale du service (`plakar-ui.<tailnet>.ts.net`), ce dernier étant lu dans une variable d'environnement. Le port n'est pas pris en compte. Le contrôle s'applique aussi aux ports d'Exploration.
- [ ] Le service refuse de démarrer si la liste des Utilisateurs autorisés est vide, avec un message explicite.
- [ ] Le service n'écoute que sur localhost. Un test vérifie qu'il ne se lie pas à toutes les interfaces.
- [ ] L'identité de l'Utilisateur autorisé est affichée sur le tableau de bord.

## Comments

**2026-11-06, revue de sécurité du ticket 01 :** sans contrôle de `Host`, une page web malveillante ouverte dans un navigateur sur la même machine que le service peut faire pointer son propre nom de domaine vers 127.0.0.1 (DNS rebinding). Elle devient alors « même origine » que le service : elle peut lire ses réponses et poser elle-même l'en-tête `Tailscale-User-Login`. Le contrôle d'identité ne suffit donc pas seul. En production, le risque ne se présente pas, puisqu'aucun navigateur ne tourne dans le réseau du conteneur partagé avec Tailscale. Il existe en dev sur le Mac, et deviendra sérieux avec la Commande de Store (06) et les Explorations (07). À vérifier pendant l'implémentation : la valeur de `Host` que `tailscale serve` transmet réellement au service.

Le critère « Le service n'écoute que sur localhost » est déjà couvert par le ticket 01 (`TestServiceRefusesToListenOutsideLocalhost`).
