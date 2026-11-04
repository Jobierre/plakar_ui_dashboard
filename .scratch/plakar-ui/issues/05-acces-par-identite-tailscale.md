# 05: Accès par identité Tailscale

**What to build:** Seuls les Utilisateurs autorisés accèdent au service. Leur identité est lue dans l'en-tête `Tailscale-User-Login`, que pose `tailscale serve` (qui supprime au passage les valeurs forgées), puis comparée à une liste fixée par variable d'environnement. Toute autre requête est refusée.

**Blocked by:** 01 (Tableau de bord minimal, de bout en bout)

**Status:** ready-for-agent

- [ ] La liste des Utilisateurs autorisés (des emails Tailscale) est lue dans une variable d'environnement.
- [ ] Une requête sans en-tête `Tailscale-User-Login` reçoit une réponse 403.
- [ ] Une requête dont l'identité n'est pas dans la liste reçoit une réponse 403.
- [ ] Le contrôle s'applique à toutes les routes : pages, actions (↻, Seuil) et toute route ajoutée plus tard.
- [ ] Le service refuse de démarrer si la liste des Utilisateurs autorisés est vide, avec un message explicite.
- [ ] Le service n'écoute que sur localhost. Un test vérifie qu'il ne se lie pas à toutes les interfaces.
- [ ] L'identité de l'Utilisateur autorisé est affichée sur le tableau de bord.
