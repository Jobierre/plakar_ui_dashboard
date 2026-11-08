# 09: Déploiement sur Unraid

**What to build:** Jobierre déploie plakar-ui dans un LXC Docker du homelab (décision du 2026-11-07 : plus directement sur Unraid, qui reste l'hôte des `plakar server`), guidé par un assistant interactif qui enchaîne les étapes que lui seul peut faire, puis vérifie depuis son PC que tout fonctionne et que l'accès est bien restreint.

**Assistant :** `scripts/deploy-wizard.sh`, à lancer sur le Mac depuis le dépôt. Il écrit `.env`, affiche les commandes pour Proxmox et le LXC, et automatise les vérifications depuis le PC (HTTPS, ports du tailnet et du LAN, 403). Le transfert du code vers le LXC reste à la main de Jobierre.

**Blocked by:** 08 (Image Docker et compose avec Tailscale)

**Status:** resolved

- [x] Le nœud `plakar-ui` apparaît sur le tailnet, sans expiration de clé (connexion par URL : la ligne `TS_AUTHKEY` du compose est commentée, choix de Jobierre).
- [x] Les dossiers `appdata` sont préparés. La config Plakar n'est lisible que par le conteneur (droits `600`).
- [x] Les Stores sont ajoutés par des Commandes de Store depuis l'interface, et chacun obtient un premier Relevé réussi.
- [x] Le tableau de bord est accessible depuis le PC à `https://plakar-ui.<tailnet>.ts.net`.
- [x] Une Exploration s'ouvre depuis le PC sur `https://plakar-ui.<tailnet>.ts.net:<port>`.
- [x] Une identité Tailscale hors de la liste est refusée.
- [x] Depuis le LAN, sans passer par Tailscale, aucun port du service ne répond.
- [x] Le LXC Docker autorise `/dev/net/tun` : le sidecar tourne en mode TUN (voir spec, « Sécurité, mode TUN obligatoire »).
- [x] Depuis un autre nœud du tailnet, `http://plakar-ui:8080` et les ports internes `plakar ui` (9400-9420) ne répondent pas : seuls 443 et la plage d'Exploration, via `tailscale serve`, sont joignables.
- [x] La politique du tailnet n'ouvre vers plakar-ui que 443 et la plage d'Exploration (exemple dans `.env.example`).

## Comments

**2026-11-08 :** déploiement fait par Jobierre dans le LXC Docker avec `scripts/deploy-wizard.sh`. Il confirme que tout fonctionne.
