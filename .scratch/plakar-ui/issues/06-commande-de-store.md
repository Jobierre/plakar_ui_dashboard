# 06: Commande de Store

**What to build:** Un bloc du tableau de bord permet de taper une Commande de Store (`plakar store add …`, `set`, `unset`, `rm`, `show`) avec la syntaxe officielle de Plakar. Le service l'exécute et affiche sa sortie. Un Store ajouté apparaît aussitôt et fait l'objet d'un Relevé, un Store retiré disparaît. Le bloc ne doit jamais pouvoir servir de terminal distant, ni laisser traîner une passphrase.

**Blocked by:** 04 (Relevés planifiés et à la demande), 05 (Accès par identité Tailscale)

**Status:** resolved

- [x] La ligne saisie est découpée en arguments, en gérant les guillemets simples et doubles (une passphrase avec des espaces fonctionne) et sans aucune expansion.
- [x] Seuls `plakar store add | set | unset | rm | show` sont acceptés, le préfixe `plakar` étant facultatif.
- [x] `store show` est accepté sans `-secrets` : plakar y masque alors les secrets (`********`). Toute forme de l'option `secrets` est refusée, à n'importe quelle position : `-secrets`, `--secrets`, `-secrets=true`, etc. (plakar lit ses options avec le paquet `flag` de Go).
- [x] Sont refusés avec un message clair : `store show` avec `-secrets`, `store import`, `store check`, `store ping`, `store ls` (qui n'existe pas dans plakar), toute autre sous-commande Plakar et toute autre commande.
- [x] Les caractères `;`, `|`, `&&`, `$(…)` et les backquotes n'ont aucun effet spécial : la commande est exécutée sans shell, en appelant directement le binaire `plakar`.
- [x] La sortie standard, la sortie d'erreur et le code de retour sont affichés.
- [x] Ni la commande ni sa sortie ne sont journalisées ou persistées. Un test vérifie qu'une passphrase saisie n'apparaît ni dans les journaux ni dans le fichier d'état.
- [x] Après une commande réussie, le Catalogue est rafraîchi. Un Store ajouté apparaît et son Relevé est lancé immédiatement. Un Store retiré disparaît, ainsi que ses données dans le fichier d'état.
- [x] Le bloc n'est accessible qu'aux Utilisateurs autorisés.

## Comments

**2026-11-06, constat pendant le ticket 01 :** `plakar store ls` n'existe pas dans plakar v1.1.6 (sous-commandes réelles : `add|check|import|ping|rm|set|show|unset`). Il faut décider ce que la Commande de Store accepte à la place de `ls`. Par exemple, `ls` pourrait être traduit en `store show` sans `-secrets`, qui masque les secrets. Il faut aussi décider du sort de `check` et `ping`, absents de la liste. À trancher avec Jobierre avant d'implémenter.

**2026-11-07, décision de Jobierre :** `ls` sort de la liste blanche. Il n'a jamais existé dans plakar, et la doc ne le mentionne pas. `store show` le remplace, mais sans `-secrets`, qui est refusé sous toutes ses formes. `check` et `ping` restent refusés : la liste blanche devient `add | set | unset | rm | show`.

**2026-11-07, réalisation :**
- La ligne est découpée comme par un shell POSIX, mais sans aucune expansion. Entre guillemets simples, tout est littéral. Entre guillemets doubles, seuls `\"` et `\\` sont échappés. Hors guillemets, `\` rend littéral le caractère suivant. Un guillemet non fermé fait refuser la commande.
- La liste blanche est appliquée après découpage. Une option globale placée avant `store` est refusée.
- `secrets` est refusé sous toutes ses formes : un ou plusieurs tirets, `=valeur`, n'importe quelle casse, à n'importe quelle position, même entre guillemets.
- La ligne saisie n'est ni journalisée, ni renvoyée dans la page. La réponse porte `Cache-Control: no-store`.
- Une Commande lancée va à son terme, rafraîchissement compris, même si le navigateur abandonne la requête. Elle est bornée à 2 min, et les Commandes sont exécutées l'une après l'autre.
- Le retrait d'un Store arrête sa planification et coupe son Relevé en cours. Chaque Relevé porte un jeton : celui d'un Store retiré n'est jamais enregistré, même si un Store du même nom est ajouté entre-temps.

**2026-11-07, risque accepté par Jobierre, après une revue de sécurité :** plakar v1.1.6 connaît la clé de Store `passphrase_cmd`, que l'on trouve dans le binaire. Elle lui fait exécuter une commande à chaque ouverture du Store, donc à chaque Relevé, et dans le conteneur. Une Commande comme `store set <store> passphrase_cmd="…"` passe donc la liste blanche et permet une exécution différée. Les clés des Commandes `add` et `set` ne sont pas filtrées : seuls les Utilisateurs autorisés accèdent au bloc. Ce choix est assumé, et la règle « jamais un terminal distant » n'est donc pas tenue face à un Utilisateur autorisé. Les options écartées étaient de refuser toute clé `*_cmd`, ou d'imposer une liste blanche de clés.
