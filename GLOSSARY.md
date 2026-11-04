# Glossaire

**Kloset**
Dépôt Plakar chiffré et dédupliqué, situé à un emplacement donné (serveur plakar, Google Drive, S3…). Contient des snapshots.

**Store**
Entrée nommée de la configuration Plakar du service, qui désigne un Kloset et les identifiants nécessaires pour l'ouvrir. Tout Store de cette configuration apparaît dans le tableau de bord (une ligne chacun).
_À éviter_ : « repo », « dépôt » pour parler de l'entrée nommée.

**Dernier snapshot**
Le snapshot le plus récent d'un Store, toutes sources confondues. Sa date est « la date de la dernière backup » du Store.

**Snapshot précédent**
Le snapshot qui précède immédiatement le Dernier snapshot dans un Store.

**Écart**
Variation de taille et de durée entre le Dernier snapshot et le Snapshot précédent. Simple indication pour l'œil : elle ne change pas l'État d'un Store.

**Relevé**
Interrogation périodique d'un Store par le service pour connaître son Dernier snapshot. Le résultat est conservé et affiché avec l'heure du Relevé. Le Store n'est pas gardé ouvert entre deux Relevés.

**Seuil de retard**
Âge maximal toléré du Dernier snapshot d'un Store. Une valeur par défaut, modifiable Store par Store.

**État d'un Store**
- _À jour_ : le Dernier snapshot est plus récent que le Seuil de retard.
- _En retard_ : le Dernier snapshot dépasse le Seuil de retard.
- _Injoignable_ : le dernier Relevé a échoué (réseau, identifiants, kloset illisible).

**Exploration**
Ouverture, à la demande, de l'interface Plakar d'un Store pour parcourir ou restaurer ses snapshots. Chaque Store a un port d'Exploration attitré et stable. L'Exploration se ferme d'elle-même après une période d'inactivité.

**Utilisateur autorisé**
Identité Tailscale figurant dans la liste d'accès du service. Seul un Utilisateur autorisé voit le tableau de bord, lance une Exploration ou gère les Stores.

**Commande de Store**
Commande `plakar store …` saisie dans l'interface pour ajouter, modifier ou retirer un Store. Elle n'est jamais conservée.
