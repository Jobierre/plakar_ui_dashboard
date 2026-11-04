# 04: Relevés planifiés et à la demande

**What to build:** Le service relève chaque Store automatiquement à intervalle régulier, ce qui maintient le tableau de bord à jour même sans visite. Chaque Store a aussi un bouton ↻ qui déclenche un Relevé immédiat. La page montre quand un Relevé est en cours, et à quelle heure a eu lieu le dernier.

**Blocked by:** 03 (État et Seuil de retard)

**Status:** ready-for-agent

- [ ] L'intervalle des Relevés vaut 1 h par défaut et peut être réglé par variable d'environnement.
- [ ] Le bouton ↻ d'un Store lance immédiatement un Relevé de ce Store.
- [ ] Deux Relevés d'un même Store ne se chevauchent jamais. Un ↻ pendant un Relevé en cours ne lance pas de second Relevé.
- [ ] Une ligne affiche « Relevé en cours » tant que le Relevé n'est pas terminé.
- [ ] Chaque ligne affiche l'heure du dernier Relevé.
- [ ] Les Relevés terminés mettent à jour le fichier d'état.
- [ ] Le Store n'est ouvert que le temps du `ls` : aucun processus `plakar` ne reste actif entre deux Relevés.
- [ ] Les tests pilotent le planificateur avec l'horloge injectée, sans attendre en temps réel.
