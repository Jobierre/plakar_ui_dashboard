# 03: État et Seuil de retard

**What to build:** Chaque Store affiche son État (À jour, En retard ou Injoignable), calculé à partir de son Dernier snapshot et de son Seuil de retard. Le Seuil par défaut se règle par variable d'environnement, et peut être modifié Store par Store dans le tableau de bord. Ce ticket introduit le fichier d'état persistant : les Seuils et le résultat du dernier Relevé y sont conservés, et survivent ainsi à un redémarrage.

**Blocked by:** 01 (Tableau de bord minimal, de bout en bout)

**Status:** resolved

- [x] Le Seuil de retard par défaut vaut 26 h et peut être réglé par variable d'environnement.
- [x] L'État est calculé à l'affichage à partir de l'horloge injectée. Un Store bascule donc En retard par le simple passage du temps, sans nouveau Relevé.
- [x] Un Store sans aucun snapshot est En retard.
- [x] Un Store dont le dernier Relevé a échoué est Injoignable. Il affiche le message d'erreur et garde ses dernières valeurs connues.
- [x] Le tableau de bord permet de modifier le Seuil d'un Store (formats lisibles comme « 26h » ou « 8d »). L'État est recalculé immédiatement, et une valeur invalide est refusée avec un message.
- [x] Les Seuils personnalisés et le résultat du dernier Relevé de chaque Store sont persistés dans un fichier d'état, à un emplacement configurable, avec une écriture atomique.
- [x] Après un redémarrage du service, le tableau de bord réaffiche les dernières valeurs connues et les Seuils avant même un nouveau Relevé.

## Comments

**2026-11-06, implémentation :**
- Variables d'environnement : `PLAKAR_UI_SEUIL` (Seuil par défaut, `26h`) et `PLAKAR_UI_STATE` (fichier d'état, `plakar-ui-state.json` dans le dossier de lancement par défaut ; le compose du ticket 08 le placera dans `appdata`). Un Seuil par défaut invalide empêche le démarrage.
- `internal/etat` : `Of` est la fonction pure (Dernier snapshot, Seuil, maintenant, erreur du Relevé) vers À jour, En retard ou Injoignable. Un âge égal au Seuil est encore À jour : seul un dépassement met En retard. `ParseSeuil` lit `26h`, `8d`, `1d12h` et toute durée Go positive ; il refuse tout signe (`1d-2h`) et tout débordement. `FormatSeuil` réécrit en jours entiers quand c'est possible (`8d`), sinon en heures (`36h` pour `1d12h`).
- Tableau de bord : colonnes « État » et « Seuil de retard » (« 26h (par défaut) » ou le Seuil propre), et un petit formulaire par ligne (`POST /stores/{store}/seuil`). Une valeur vide rend le Seuil par défaut. Une valeur invalide renvoie 400 et la page avec le message sur la ligne du Store, et rien ne change. Tant qu'aucun Relevé n'a abouti, la colonne État reste vide.
- Le formulaire est protégé par `http.CrossOriginProtection` : une page tierce ouverte dans le même navigateur ne peut pas le poster (test `TestSeuilFormIsRefusedFromAnotherSite`). Ce n'était pas demandé, mais c'est la première action en écriture du service, et le ticket 05 ne couvre que l'en-tête Host.
- Fichier d'état (`internal/catalogue/state.go`) : du JSON, avec par Store son Seuil propre et le résultat de son dernier Relevé (l'erreur y est un texte). Écriture atomique : fichier temporaire dans le même dossier, fsync, rename, puis fsync du dossier. Un fichier illisible empêche le démarrage, pour ne jamais écraser des Seuils qu'on n'a pas su lire. L'état d'un Store sorti de la config plakar est ignoré au chargement, et disparaît à l'écriture suivante. Si l'écriture d'un Seuil échoue, le Seuil n'est pas changé et la page affiche l'erreur.
- Un Relevé en échec garde le Dernier snapshot, le Snapshot précédent et l'Écart du Relevé précédent. L'heure du Relevé est celle de l'échec.
- Faux plakar : le champ `delay_ms` fait attendre une réponse. Le test de redémarrage s'en sert pour prouver que les valeurs sont réaffichées avant tout nouveau Relevé.
- Le critère du ticket 02 « L'Écart n'a aucune influence sur l'État » est maintenant testé (`TestEcartNeverChangesTheEtat`).
- À trancher plus tard (`/domain-modeling`) : « fichier d'état » emploie « état » dans un autre sens que l'« État d'un Store » du glossaire.
