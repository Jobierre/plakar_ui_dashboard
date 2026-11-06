# 02: Écart avec le Snapshot précédent, et test de contrat

**What to build:** Le Relevé lit les deux derniers snapshots d'un Store. Le tableau de bord affiche alors l'Écart de taille et de durée entre le Dernier snapshot et le Snapshot précédent, ce qui permet de repérer par exemple une source vide. Un test de contrat, lancé avec le vrai `plakar`, garantit que la lecture des sorties reste correcte pour la version de Plakar figée.

**Blocked by:** 01 (Tableau de bord minimal, de bout en bout)

**Status:** resolved

- [x] Le Relevé garde les deux premières lignes de `ls -tags`, Plakar triant du plus récent au plus ancien.
- [x] L'Écart de taille est affiché en pourcentage signé (par exemple « −97 % »), à côté de la taille.
- [x] L'Écart de durée est affiché à côté de la durée.
- [x] Un Store qui n'a qu'un seul snapshot n'affiche aucun Écart.
- [ ] L'Écart n'a aucune influence sur l'État d'un Store.
- [x] Le test de contrat crée un Kloset jetable dans un dossier temporaire, y fait deux backups d'un petit dossier en modifiant le contenu entre les deux, puis vérifie via le service le Dernier snapshot, le Snapshot précédent et l'Écart.
- [x] Le test de contrat peut être sauté proprement quand `plakar` n'est pas installé, et il est clairement identifié comme tel.

## Comments

**2026-11-06, implémentation :**
- Décisions de Jobierre : l'Écart de durée est une différence signée (« +4s », « −1m3s », « 0s » sans signe). Si le Snapshot précédent pèse 0 B, l'Écart de taille affiche « nouveau » (« 0 % » si les deux pèsent 0 B).
- Le Relevé produit `Previous` et `Ecart` (`internal/releve`) ; la mise en forme est dans `internal/web`. Deux colonnes « Écart de taille » et « Écart de durée » suivent la taille et la durée ; leur infobulle désigne le Snapshot précédent (ID, taille, durée).
- L'Écart de taille est calculé sur les tailles telles que plakar les affiche (arrondies à une décimale) : c'est une indication pour l'œil, pas une mesure exacte.
- Critère « aucune influence sur l'État » : l'État n'existe pas encore. Rien ne lit l'Écart pour décider d'un État ; le test viendra avec le ticket 03.
- Test de contrat : `internal/app/contract_test.go` (`TestContractWithRealPlakar`). Il prend le `plakar` du PATH, ou `PLAKAR_UI_CONTRACT_PLAKAR`, journalise sa version, et se saute avec un message explicite s'il est introuvable. Il ne vérifie pas encore que la version est celle figée : à brancher sur la version de l'image au ticket 08.
- Piège rencontré : plakar ouvre le socket Unix de son démon `cached` dans `-cachedir` ; sous `t.TempDir()` le chemin dépasse 104 octets (limite macOS) et plakar répond « failed to run cached ». Le cache du test est donc dans un dossier court sous `/tmp`, supprimé en fin de test.
