# 09: Déploiement sur Unraid

**What to build:** Jobierre déploie plakar-ui sur Unraid, guidé par un assistant interactif (`/wizard`) qui enchaîne les étapes que lui seul peut faire, puis vérifie depuis son PC que tout fonctionne et que l'accès est bien restreint.

**Blocked by:** 08 (Image Docker et compose avec Tailscale)

**Status:** ready-for-human

- [ ] Une clé d'authentification Tailscale est créée et renseignée, et le nœud `plakar-ui` apparaît sur le tailnet.
- [ ] Les dossiers `appdata` sont préparés. La config Plakar n'est lisible que par le conteneur (droits `600`).
- [ ] Les Stores sont ajoutés par des Commandes de Store depuis l'interface, et chacun obtient un premier Relevé réussi.
- [ ] Le tableau de bord est accessible depuis le PC à `https://plakar-ui.<tailnet>.ts.net`.
- [ ] Une Exploration s'ouvre depuis le PC sur `https://plakar-ui.<tailnet>.ts.net:<port>`.
- [ ] Une identité Tailscale hors de la liste est refusée.
- [ ] Depuis le LAN, sans passer par Tailscale, aucun port du service ne répond.
