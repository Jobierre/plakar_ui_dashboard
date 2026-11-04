# 08: Image Docker et compose avec Tailscale

**What to build:** Le service est empaqueté dans une image avec une version de `plakar` figée. Un compose le déploie à côté d'un sidecar Tailscale qui l'expose sur le tailnet en HTTPS : le tableau de bord sur 443, et chaque port de la plage d'Exploration. Aucun port n'est publié sur le LAN. La config Plakar, le cache Plakar et le fichier d'état vivent dans des volumes persistants.

**Blocked by:** 06 (Commande de Store), 07 (Explorations)

**Status:** ready-for-agent

- [ ] L'image se construit en plusieurs étapes : elle contient le binaire du service et `plakar`, dans une version figée et explicite. Elle ne contient pas rclone.
- [ ] Le test de contrat (ticket 02) s'exécute avec le `plakar` de l'image.
- [ ] Le compose définit un sidecar Tailscale et le service, qui partage son réseau. Aucune directive ne publie de port sur l'hôte.
- [ ] La configuration `tailscale serve` est déclarative : HTTPS 443 vers le tableau de bord, et chaque port de la plage vers le même port local.
- [ ] Trois volumes persistants sont prévus : la config Plakar, le cache Plakar et le fichier d'état.
- [ ] Le nom `unraid` est résolu depuis le conteneur.
- [ ] Un exemple de fichier d'environnement documente toutes les variables : la clé Tailscale, les Utilisateurs autorisés, le nom d'hôte public, le Seuil par défaut, l'intervalle de Relevé, l'inactivité d'Exploration et la plage de ports.
- [ ] Vérification locale : l'image démarre, et le service répond 403 sans en-tête d'identité depuis l'intérieur du réseau du conteneur.
