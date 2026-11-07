# 04: Relevés planifiés et à la demande

**What to build:** Le service relève chaque Store automatiquement à intervalle régulier, ce qui maintient le tableau de bord à jour même sans visite. Chaque Store a aussi un bouton ↻ qui déclenche un Relevé immédiat. La page montre quand un Relevé est en cours, et à quelle heure a eu lieu le dernier.

**Blocked by:** 03 (État et Seuil de retard)

**Status:** resolved

- [x] L'intervalle des Relevés vaut 1 h par défaut et peut être réglé par variable d'environnement.
- [x] Le bouton ↻ d'un Store lance immédiatement un Relevé de ce Store.
- [x] Deux Relevés d'un même Store ne se chevauchent jamais. Un ↻ pendant un Relevé en cours ne lance pas de second Relevé.
- [x] Une ligne affiche « Relevé en cours » tant que le Relevé n'est pas terminé.
- [x] Chaque ligne affiche l'heure du dernier Relevé.
- [x] Les Relevés terminés mettent à jour le fichier d'état.
- [x] Le Store n'est ouvert que le temps du `ls` : aucun processus `plakar` ne reste actif entre deux Relevés.
- [x] Les tests pilotent le planificateur avec l'horloge injectée, sans attendre en temps réel.

## Comments

Implémenté le 2026-11-07.

- Planificateur (`internal/planificateur`) : une boucle par Store. Premier Relevé au démarrage, puis un à chaque intervalle (`PLAKAR_UI_INTERVALLE`, durée Go, `1h` par défaut ; une valeur illisible ou ≤ 0 empêche le démarrage). L'intervalle se compte d'un début de Relevé au suivant. Une échéance qui tombe pendant un Relevé en cours est sautée. Le parallélisme (4) et le délai maximal d'un Relevé (10 min) restent des constantes.
- Non-chevauchement : `Catalogue.Begin` marque le Store « en cours » de façon atomique, `Record` (ou `Abandon`) lève la marque. Un ↻ pendant un Relevé ne fait rien, et le bouton est désactivé. La marque couvre aussi l'attente du sémaphore : une ligne en file d'attente affiche donc déjà « Relevé en cours ».
- Bouton ↻ : `POST /stores/{store}/releve` → 303 vers `/`. Il renvoie 404 pour un Store inconnu, et 403 depuis un autre site (même protection cross-origin que le Seuil).
- Affichage : colonnes « Dernier Relevé » (heure locale, champ `releve`) et « Relevé en cours » (champ `releve-en-cours`). Le message d'une ligne jamais relevée devient « Aucun Relevé terminé ».
- Un Relevé coupé par l'arrêt du service n'est plus enregistré. Avant, il était noté en échec (« context canceled »), et le Store paraissait Injoignable au redémarrage. Un Relevé qui a abouti juste avant l'arrêt est, lui, enregistré.
- Horloge injectée : `Options.After` (`time.After` en production). L'horloge des tests ne termine ses attentes que sur `advance`.
- Faux plakar : il note son PID dans `pids`. Le test vérifie qu'aucun processus `plakar` n'existe encore après les Relevés.
- `TestStoreFallsBehind…` tourne maintenant avec un intervalle de 1000 h, pour qu'aucun Relevé planifié ne tombe pendant le test.
- Pour le ticket 06 : `Planificateur.Start` fige la liste des Stores planifiés. Un Store ajouté pourra être relevé tout de suite par `Trigger`, mais il faudra une méthode `Add` pour le planifier ensuite toutes les heures. Pour un Store retiré, sa boucle s'arrête déjà d'elle-même (`ErrUnknownStore`).
- Limites connues des tests : l'assertion « pas de Relevé à 59 min » est faible, faute de point de synchronisation. Le test « Relevé en cours » s'appuie sur un délai réel de 500 ms du faux plakar.
