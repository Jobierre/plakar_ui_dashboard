# plakar-ui

Tableau de bord léger, multi-Stores, pour [Plakar](https://github.com/PlakarKorp/plakar).

Une page, une ligne par Store : la date de son **Dernier snapshot**, son **État**
(_À jour_, _En retard_, _Injoignable_) et l'**Écart** avec le snapshot précédent.
Le service ne fait pas de sauvegarde : il **relève** périodiquement chaque Store
et dit, d'un coup d'œil, si les sauvegardes tournent.

Le vocabulaire (Store, Relevé, Seuil de retard, Groupe, Focus…) est défini dans
[`GLOSSARY.md`](GLOSSARY.md).

## Fonctionnalités

- **Relevé** planifié de chaque Store (toutes les heures par défaut), ou à la
  demande : un Store, un Groupe, ou tous les Stores.
- **État** calculé à l'affichage à partir d'un **Seuil de retard** (26h par
  défaut), modifiable Store par Store.
- **Groupes** : ranger les Stores par glisser-déposer, replier un Groupe, ouvrir
  le **Focus** d'un Groupe (une page à mettre en favori).
- **Stores disparus** : un Store retiré de la config Plakar garde sa place dans
  son Groupe tant qu'on ne l'en retire pas.
- Vue **Simple** ou **Détaillée**.
- **Exploration** : ouvre à la demande le `plakar ui` d'un Store pour parcourir ou
  restaurer ses snapshots, sur un port attitré ; elle s'arrête seule après 15
  minutes d'inactivité.
- **Commande de Store** : ajouter, modifier ou retirer un Store depuis
  l'interface. Seules `plakar store add | set | unset | rm | show` sont
  acceptées, sans shell, et l'option `-secrets` est refusée. Une commande n'est
  jamais journalisée ni conservée (elle peut contenir une passphrase).

## Architecture

```
config Plakar (stores.yml)
        │
        ▼
   Catalogue ──► Relevé (sous-processus plakar, sans shell)
                     │
                     ▼
         fichier d'état JSON (écriture atomique, pas de base de données)
                     │
                     ▼
              État + Web (templates HTML embarqués)
                     │
                     ▼
     127.0.0.1:8080 ◄── tailscale serve (HTTPS, identité Tailscale)
```

- Un seul binaire Go, sans dépendance à l'exécution hormis `plakar`.
- `plakar` est **figé** dans l'image (`PLAKAR_VERSION` du `Dockerfile`). Une
  montée de version se fait là, et seulement là.
- Le service n'écoute **que sur localhost**. On l'atteint par le tailnet, via un
  sidecar Tailscale qui partage son réseau.

### Arborescence

| Chemin | Rôle |
|---|---|
| `cmd/plakar-ui` | Point d'entrée |
| `internal/app` | Assemblage, configuration, tests de bout en bout |
| `internal/plakar` | Lancement du binaire plakar |
| `internal/catalogue` | Stores connus, Groupes, ports d'Exploration, état persisté |
| `internal/releve` / `internal/planificateur` | Relevés et leur planification |
| `internal/etat` | Calcul de l'État et du Seuil de retard |
| `internal/exploration` | Cycle de vie des `plakar ui` |
| `internal/commande` | Commandes de Store |
| `internal/acces` | Contrôle d'accès (Utilisateurs autorisés, en-tête `Host`) |
| `internal/tailscaleserve` | Écriture de la config `tailscale serve` |
| `internal/web` | Routes HTTP, templates, CSS/JS |
| `internal/fakeplakar` | Faux binaire plakar pour les tests |
| `scripts/deploy-wizard.sh` | Assistant de déploiement pas à pas |

## Sécurité

- **Accès** : l'identité vient de l'en-tête `Tailscale-User-Login`, posé par
  `tailscale serve`. Seules les identités de `PLAKAR_UI_UTILISATEURS` passent,
  les autres reçoivent 403.
- **DNS rebinding** : seules les requêtes adressées à `PLAKAR_UI_HOTE` ou à
  localhost sont servies.
- **Sidecar en mode TUN** (`TS_USERSPACE=false`) : en userspace, un pair du
  tailnet pourrait joindre directement le port 8080 sans passer par `serve`, et
  forger l'en-tête d'identité.
- **Défense en profondeur** : dans la politique du tailnet, n'ouvrir vers le
  nœud que `tcp:443` et la plage d'Exploration (voir `.env.example`).
- La config Plakar contient les passphrases : dossier en `700`.

## Déploiement (Docker Compose)

Cible prévue : un LXC Docker sous Proxmox, les Stores étant servis par des
`plakar server` sur Unraid.

L'assistant `scripts/deploy-wizard.sh` déroule toutes les étapes (tailnet,
`/dev/net/tun` dans le LXC, dossiers appdata, construction, vérifications).
En résumé :

```sh
# 1. Configuration
cp .env.example .env    # puis remplir

# 2. Dossiers persistants, à l'utilisateur 1000:1000 de l'image
#    (sinon Docker les crée en root)
mkdir -p $PLAKAR_UI_APPDATA/{tailscale,serve,plakar-config,plakar-cache,state}
chown 1000:1000 $PLAKAR_UI_APPDATA/{serve,plakar-config,plakar-cache,state}
chmod 700 $PLAKAR_UI_APPDATA/plakar-config

# 3. Construction et démarrage
docker compose up -d --build

# 4. Connexion du nœud au tailnet (sans TS_AUTHKEY) : suivre l'URL affichée
docker compose logs tailscale
```

Le tableau de bord est ensuite sur `https://plakar-ui.<tailnet>.ts.net`.

> Dans un LXC Proxmox, `/dev/net/tun` doit être autorisé dans la conf du
> conteneur, sans quoi le sidecar ne démarre pas en mode TUN.

### Construction de l'image

Le `Dockerfile` compile plakar à la version figée, puis le service, puis lance
le **test de contrat** contre ce plakar. L'image n'existe que si ce test passe :
un changement du format des sorties de plakar casse la construction, pas la
production.

## Configuration

Variables d'environnement (voir `.env.example`) :

| Variable | Défaut | Rôle |
|---|---|---|
| `PLAKAR_UI_UTILISATEURS` | — (requis) | Identités Tailscale autorisées, séparées par des virgules |
| `PLAKAR_UI_HOTE` | — (requis) | Nom du service sur le tailnet, ex. `plakar-ui.<tailnet>.ts.net` |
| `PLAKAR_UI_SEUIL` | `26h` | Seuil de retard par défaut (`26h`, `8d`, `1d12h`…) |
| `PLAKAR_UI_INTERVALLE` | `1h` | Intervalle entre deux Relevés planifiés |
| `PLAKAR_UI_INACTIVITE` | `15m` | Inactivité avant arrêt d'une Exploration |
| `PLAKAR_UI_PORTS` | `9300-9320` | Plage des ports publics d'Exploration |
| `PLAKAR_UI_DECALAGE` | `100` | Décalage port public → port interne du `plakar ui` (doit dépasser la largeur de la plage) |
| `PLAKAR_UI_ADDR` | `127.0.0.1:8080` | Adresse d'écoute, obligatoirement sur localhost |
| `PLAKAR_UI_PLAKAR_BIN` | `plakar` | Chemin du binaire plakar |
| `PLAKAR_UI_STATE` | `plakar-ui-state.json` | Fichier d'état (Seuils, Groupes, derniers Relevés) |
| `PLAKAR_UI_SERVE_CONFIG` | _(vide)_ | Config `tailscale serve` écrite au démarrage (rien si vide) |

Côté compose seulement : `PLAKAR_UI_APPDATA` (dossier des volumes),
`UNRAID_IP` (résolution du nom `unraid` dans le conteneur) et `TS_AUTHKEY`
(facultative).

## Développement

Prérequis : Go 1.27.

```sh
go build ./...
go test ./...
```

Les tests utilisent `internal/fakeplakar`, qui rejoue des sorties enregistrées
de plakar. Le test de contrat (`go test -run Contract ./internal/app/`) tourne
contre un vrai plakar : il est sauté si plakar n'est pas installé, sauf si
`PLAKAR_UI_CONTRACT_PLAKAR` désigne un binaire, auquel cas il est obligatoire.

Lancement local :

```sh
PLAKAR_UI_UTILISATEURS=moi@example.com go run ./cmd/plakar-ui
# hors tailscale serve, l'identité doit être posée à la main :
curl -H 'Tailscale-User-Login: moi@example.com' http://127.0.0.1:8080/
```

## Documentation du projet

- [`GLOSSARY.md`](GLOSSARY.md) : vocabulaire du domaine.
- `.scratch/<feature>/spec.md` et `.scratch/<feature>/issues/` : specs et
  tickets (suivi en fichiers markdown locaux).
- `docs/agents/` : conventions pour les agents (tickets, triage, domaine).
