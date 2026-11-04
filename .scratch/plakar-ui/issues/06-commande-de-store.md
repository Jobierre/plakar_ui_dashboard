# 06: Commande de Store

**What to build:** Un bloc du tableau de bord permet de taper une Commande de Store (`plakar store add …`, `set`, `unset`, `rm`, `ls`) avec la syntaxe officielle de Plakar. Le service l'exécute et affiche sa sortie. Un Store ajouté apparaît aussitôt et fait l'objet d'un Relevé, un Store retiré disparaît. Le bloc ne doit jamais pouvoir servir de terminal distant, ni laisser traîner une passphrase.

**Blocked by:** 04 (Relevés planifiés et à la demande), 05 (Accès par identité Tailscale)

**Status:** ready-for-agent

- [ ] La ligne saisie est découpée en arguments, en gérant les guillemets simples et doubles (une passphrase avec des espaces fonctionne) et sans aucune expansion.
- [ ] Seuls `plakar store add | set | unset | rm | ls` sont acceptés, le préfixe `plakar` étant facultatif.
- [ ] Sont refusés avec un message clair : `store show`, `store import`, toute autre sous-commande Plakar et toute autre commande.
- [ ] Les caractères `;`, `|`, `&&`, `$(…)` et les backquotes n'ont aucun effet spécial : la commande est exécutée sans shell, en appelant directement le binaire `plakar`.
- [ ] La sortie standard, la sortie d'erreur et le code de retour sont affichés.
- [ ] Ni la commande ni sa sortie ne sont journalisées ou persistées. Un test vérifie qu'une passphrase saisie n'apparaît ni dans les journaux ni dans le fichier d'état.
- [ ] Après une commande réussie, le Catalogue est rafraîchi. Un Store ajouté apparaît et son Relevé est lancé immédiatement. Un Store retiré disparaît, ainsi que ses données dans le fichier d'état.
- [ ] Le bloc n'est accessible qu'aux Utilisateurs autorisés.
