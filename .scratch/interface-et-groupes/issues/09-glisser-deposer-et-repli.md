# 09: Glisser-déposer et repli des Groupes

**What to build:** Jobierre range ses Stores et ordonne ses Groupes à la souris, par glisser-déposer, dans les deux vues. Il replie les Groupes qui vont bien, et son navigateur s'en souvient (spec : stories 42 à 44, 49, 50, 68).

**Blocked by:** 04 (Vue simple et bascule), 06 (Groupes, ordre, renommage et suppression)

**Status:** ready-for-agent

- [ ] Glisser un Store dans un autre Groupe ou à une autre place de son Groupe, dans la vue détaillée (sections) comme dans la vue simple (colonnes de tuiles). Chaque dépôt envoie le même `POST` que le rangement avec position du ticket 06, puis recharge la page.
- [ ] Glisser un Groupe par sa poignée ⠿ change l'ordre des Groupes ; « Sans groupe » ne se glisse pas et reste en dernier.
- [ ] Le chevron ▾ replie ou déplie un Groupe ; l'état replié est retenu dans `localStorage`, pas par le service.
- [ ] Les quatre gestes de l'en-tête sont distincts : poignée pour glisser, nom pour le focus, chevron pour replier, menu ⋯ pour renommer ou supprimer.
- [ ] Le script utilise l'API HTML5 native, sans bibliothèque, et n'insère jamais de nom comme du HTML.
- [ ] Vérification à la main dans le navigateur, contre le prototype (variante D) : pas de test automatique du JavaScript.

## Comments
