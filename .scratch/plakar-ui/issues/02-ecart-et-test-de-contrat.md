# 02: Écart avec le Snapshot précédent, et test de contrat

**What to build:** Le Relevé lit les deux derniers snapshots d'un Store. Le tableau de bord affiche alors l'Écart de taille et de durée entre le Dernier snapshot et le Snapshot précédent, ce qui permet de repérer par exemple une source vide. Un test de contrat, lancé avec le vrai `plakar`, garantit que la lecture des sorties reste correcte pour la version de Plakar figée.

**Blocked by:** 01 (Tableau de bord minimal, de bout en bout)

**Status:** ready-for-agent

- [ ] Le Relevé garde les deux premières lignes de `ls -tags`, Plakar triant du plus récent au plus ancien.
- [ ] L'Écart de taille est affiché en pourcentage signé (par exemple « −97 % »), à côté de la taille.
- [ ] L'Écart de durée est affiché à côté de la durée.
- [ ] Un Store qui n'a qu'un seul snapshot n'affiche aucun Écart.
- [ ] L'Écart n'a aucune influence sur l'État d'un Store.
- [ ] Le test de contrat crée un Kloset jetable dans un dossier temporaire, y fait deux backups d'un petit dossier en modifiant le contenu entre les deux, puis vérifie via le service le Dernier snapshot, le Snapshot précédent et l'Écart.
- [ ] Le test de contrat peut être sauté proprement quand `plakar` n'est pas installé, et il est clairement identifié comme tel.
