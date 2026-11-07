# 08: Image Docker et compose avec Tailscale

**What to build:** Le service est empaqueté dans une image avec une version de `plakar` figée. Un compose le déploie à côté d'un sidecar Tailscale qui l'expose sur le tailnet en HTTPS : le tableau de bord sur 443, et chaque port de la plage d'Exploration. Aucun port n'est publié sur le LAN. La config Plakar, le cache Plakar et le fichier d'état vivent dans des volumes persistants.

**Blocked by:** 06 (Commande de Store), 07 (Explorations)

**Status:** resolved

- [x] L'image se construit en plusieurs étapes : elle contient le binaire du service et `plakar`, dans une version figée et explicite. Elle ne contient pas rclone.
- [x] Le test de contrat (ticket 02) s'exécute avec le `plakar` de l'image.
- [x] Le compose définit un sidecar Tailscale et le service, qui partage son réseau. Aucune directive ne publie de port sur l'hôte.
- [x] La configuration `tailscale serve` est déclarative : HTTPS 443 vers le tableau de bord, et chaque port de la plage vers le même port local.
- [x] Trois volumes persistants sont prévus : la config Plakar, le cache Plakar et le fichier d'état.
- [x] Le nom `unraid` est résolu depuis le conteneur.
- [x] Un exemple de fichier d'environnement documente toutes les variables : la clé Tailscale, les Utilisateurs autorisés, le nom d'hôte public, le Seuil par défaut, l'intervalle de Relevé, l'inactivité d'Exploration et la plage de ports.
- [x] Vérification locale : l'image démarre, et le service répond 403 sans en-tête d'identité depuis l'intérieur du réseau du conteneur.

## Comments

**2026-11-07, réalisation :**
- Image `Dockerfile` en quatre étages : `plakar` (v1.1.6 compilé depuis ses sources), `build`, `contract` (test de contrat obligatoire avec ce plakar, `PLAKAR_UI_CONTRACT_PLAKAR`), et l'image finale `alpine:3.24`, 89 Mo, UID 1000, sans rclone. Elle copie le binaire depuis l'étage `contract` : sans contrat réussi, pas d'image.
- `serve.json` est généré au démarrage par le service (`PLAKAR_UI_SERVE_CONFIG`, paquet `internal/tailscaleserve`) depuis les adresses réelles d'écoute, et non figé. Le sidecar surveille le dossier et réapplique. Un fichier illisible ferait s'arrêter le sidecar : l'écriture est atomique.
- Compose : `tailscale/tailscale:v1.104.4` en mode TUN noyau, `network_mode: service:tailscale`, aucun `ports:`, volumes en syntaxe courte sous `${PLAKAR_UI_APPDATA}` (config, cache, état, plus `serve` et l'état tailscale), `unraid` vers `UNRAID_IP`. `TS_AUTHKEY` est commenté par choix de Jobierre.
- Deux défauts trouvés par le test de contrat sous Linux et corrigés à la source (voir spec, « Vérifié au ticket 08 ») : l'Exploration qui donnait son lien avant l'écoute de `plakar ui`, et la bannière de premier lancement de plakar.
- Vérification locale (OrbStack) : 403 sans en-tête d'identité sur 8080 et 9300, 200 avec un Utilisateur autorisé, `serve.json` avec 22 ports HTTPS, `unraid` résolu depuis plakar-ui sous compose, arrêt propre par SIGTERM.
- Revue de sécurité après commit : le mode userspace, choisi d'abord pour le LXC, permettait de contourner l'Accès (voir spec, « Sécurité, mode TUN obligatoire »). Le sidecar est passé en TUN (`/dev/net/tun`, `NET_ADMIN`), et une règle ACL du tailnet est documentée. Ticket 09 : autoriser `/dev/net/tun` dans le LXC.
