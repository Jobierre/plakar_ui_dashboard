# 06: Commande de Store

**What to build:** Un bloc du tableau de bord permet de taper une Commande de Store (`plakar store add …`, `set`, `unset`, `rm`, `show`) avec la syntaxe officielle de Plakar. Le service l'exécute et affiche sa sortie. Un Store ajouté apparaît aussitôt et fait l'objet d'un Relevé, un Store retiré disparaît. Le bloc ne doit jamais pouvoir servir de terminal distant, ni laisser traîner une passphrase.

**Blocked by:** 04 (Relevés planifiés et à la demande), 05 (Accès par identité Tailscale)

**Status:** ready-for-agent

- [ ] La ligne saisie est découpée en arguments, en gérant les guillemets simples et doubles (une passphrase avec des espaces fonctionne) et sans aucune expansion.
- [ ] Seuls `plakar store add | set | unset | rm | show` sont acceptés, le préfixe `plakar` étant facultatif.
- [ ] `store show` est accepté sans `-secrets` : plakar y masque alors les secrets (`********`). Toute forme de l'option `secrets` est refusée, à n'importe quelle position : `-secrets`, `--secrets`, `-secrets=true`, etc. (plakar lit ses options avec le paquet `flag` de Go).
- [ ] Sont refusés avec un message clair : `store show` avec `-secrets`, `store import`, `store check`, `store ping`, `store ls` (qui n'existe pas dans plakar), toute autre sous-commande Plakar et toute autre commande.
- [ ] Les caractères `;`, `|`, `&&`, `$(…)` et les backquotes n'ont aucun effet spécial : la commande est exécutée sans shell, en appelant directement le binaire `plakar`.
- [ ] La sortie standard, la sortie d'erreur et le code de retour sont affichés.
- [ ] Ni la commande ni sa sortie ne sont journalisées ou persistées. Un test vérifie qu'une passphrase saisie n'apparaît ni dans les journaux ni dans le fichier d'état.
- [ ] Après une commande réussie, le Catalogue est rafraîchi. Un Store ajouté apparaît et son Relevé est lancé immédiatement. Un Store retiré disparaît, ainsi que ses données dans le fichier d'état.
- [ ] Le bloc n'est accessible qu'aux Utilisateurs autorisés.

## Comments

**2026-11-06, constat pendant le ticket 01 :** `plakar store ls` n'existe pas dans plakar v1.1.6 (sous-commandes réelles : `add|check|import|ping|rm|set|show|unset`). Il faut décider ce que la Commande de Store accepte à la place de `ls`. Par exemple, `ls` pourrait être traduit en `store show` sans `-secrets`, qui masque les secrets. Il faut aussi décider du sort de `check` et `ping`, absents de la liste. À trancher avec Jobierre avant d'implémenter.

**2026-11-07, décision de Jobierre :** `ls` sort de la liste blanche. Il n'a jamais existé dans plakar, et la doc ne le mentionne pas. `store show` le remplace, mais sans `-secrets`, qui est refusé sous toutes ses formes. `check` et `ping` restent refusés : la liste blanche devient `add | set | unset | rm | show`.
