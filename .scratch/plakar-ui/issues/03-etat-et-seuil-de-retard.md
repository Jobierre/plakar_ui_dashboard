# 03: État et Seuil de retard

**What to build:** Chaque Store affiche son État (À jour, En retard ou Injoignable), calculé à partir de son Dernier snapshot et de son Seuil de retard. Le Seuil par défaut se règle par variable d'environnement, et peut être modifié Store par Store dans le tableau de bord. Ce ticket introduit le fichier d'état persistant : les Seuils et le résultat du dernier Relevé y sont conservés, et survivent ainsi à un redémarrage.

**Blocked by:** 01 (Tableau de bord minimal, de bout en bout)

**Status:** ready-for-agent

- [ ] Le Seuil de retard par défaut vaut 26 h et peut être réglé par variable d'environnement.
- [ ] L'État est calculé à l'affichage à partir de l'horloge injectée. Un Store bascule donc En retard par le simple passage du temps, sans nouveau Relevé.
- [ ] Un Store sans aucun snapshot est En retard.
- [ ] Un Store dont le dernier Relevé a échoué est Injoignable. Il affiche le message d'erreur et garde ses dernières valeurs connues.
- [ ] Le tableau de bord permet de modifier le Seuil d'un Store (formats lisibles comme « 26h » ou « 8d »). L'État est recalculé immédiatement, et une valeur invalide est refusée avec un message.
- [ ] Les Seuils personnalisés et le résultat du dernier Relevé de chaque Store sont persistés dans un fichier d'état, à un emplacement configurable, avec une écriture atomique.
- [ ] Après un redémarrage du service, le tableau de bord réaffiche les dernières valeurs connues et les Seuils avant même un nouveau Relevé.
