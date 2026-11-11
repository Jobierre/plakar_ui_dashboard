package app_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// Ordre des Groupes et des Stores, renommage et suppression d'un Groupe.

// deplacerStore appuie sur ↑ (« haut ») ou ↓ (« bas ») dans le dépliage d'un
// Store.
func (s *service) deplacerStore(store, sens string) (int, string) {
	s.t.Helper()
	return s.postForm("/stores/"+url.PathEscape(store)+"/ordre", url.Values{"sens": {sens}})
}

func TestStoreMovesUpAndDownWithinItsGroupe(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")
	for _, s := range []string{"arr", "immich", "zeta"} {
		svc.ranger(s, "Unraid")
	}

	svc.deplacerStore("zeta", "haut")
	code, body := svc.deplacerStore("arr", "bas")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	if got, want := rangement(groupes(t, body))["Unraid"], []string{"zeta", "arr", "immich"}; !slices.Equal(got, want) {
		t.Errorf("Unraid = %q, attendu %q", got, want)
	}
}

func TestStoreMovesWithinSansGroupeAndStopsAtTheEnds(t *testing.T) {
	svc, _ := troisStores(t, nil)

	svc.deplacerStore("arr", "haut")
	svc.deplacerStore("zeta", "bas")
	_, body := svc.deplacerStore("zeta", "haut")

	if got, want := rangement(groupes(t, body))["Sans groupe"], []string{"arr", "zeta", "immich"}; !slices.Equal(got, want) {
		t.Errorf("Sans groupe = %q, attendu %q", got, want)
	}
}

func TestMovingAnUnknownStoreOrInAnUnknownSensChangesNothing(t *testing.T) {
	svc, _ := troisStores(t, nil)

	for _, c := range []struct {
		store, sens string
		code        int
	}{{"inconnu", "haut", http.StatusNotFound}, {"zeta", "gauche", http.StatusBadRequest}} {
		code, body := svc.deplacerStore(c.store, c.sens)

		if code != c.code {
			t.Errorf("%s %s : code %d, attendu %d", c.store, c.sens, code, c.code)
		}
		assertRangementErreur(t, body)
	}
	_, body := svc.get("/")
	if got, want := rangement(groupes(t, body))["Sans groupe"], []string{"arr", "immich", "zeta"}; !slices.Equal(got, want) {
		t.Errorf("Sans groupe = %q, attendu %q", got, want)
	}
}

// assertRangementErreur vérifie que la page porte un message de refus de
// rangement.
func assertRangementErreur(t *testing.T, body string) {
	t.Helper()
	if el := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-rangement-erreur"); return ok }); el == nil || text(el) == "" {
		t.Errorf("aucun message de refus :\n%s", body)
	}
}

// placer range un Store dans ce Groupe, avant ce Store (à la fin si avant est
// vide) : l'action que réutilise le glisser-déposer.
func (s *service) placer(store, groupe, avant string) (int, string) {
	s.t.Helper()
	return s.postForm("/stores/"+url.PathEscape(store)+"/groupe", url.Values{"groupe": {groupe}, "avant": {avant}})
}

func TestStoreIsPlacedBeforeAnotherInItsNewOrItsOwnGroupe(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")
	svc.ranger("arr", "Unraid")
	svc.ranger("immich", "Unraid")

	// Dans un autre Groupe, avant un Store.
	svc.placer("zeta", "Unraid", "immich")
	// Dans son propre Groupe, avant un Store, puis à la fin.
	svc.placer("immich", "Unraid", "arr")
	code, body := svc.placer("arr", "Unraid", "")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	want := map[string][]string{"Unraid": {"immich", "zeta", "arr"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
	// Avant lui-même : le Store garde sa place.
	_, body = svc.placer("zeta", "Unraid", "zeta")
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
}

func TestPlacingBeforeAStoreOutsideTheTargetGroupeChangesNothing(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")
	svc.ranger("arr", "Unraid")

	for _, avant := range []string{"immich", "inconnu"} {
		code, body := svc.placer("zeta", "Unraid", avant)

		if code != http.StatusNotFound {
			t.Errorf("avant %s : code %d, attendu %d", avant, code, http.StatusNotFound)
		}
		assertRangementErreur(t, body)
	}
	_, body := svc.get("/")
	want := map[string][]string{"Unraid": {"arr"}, "Sans groupe": {"immich", "zeta"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
}

// deplacerGroupe appuie sur ↑ (« haut ») ou ↓ (« bas ») de l'en-tête d'un
// Groupe.
func (s *service) deplacerGroupe(groupe, sens string) (int, string) {
	s.t.Helper()
	return s.postForm("/groupes/"+url.PathEscape(groupe)+"/ordre", url.Values{"sens": {sens}})
}

// placerGroupe déplace un Groupe avant un autre (ou avant « Sans groupe »,
// c'est-à-dire en dernier) : l'action que réutilise le glisser-déposer.
func (s *service) placerGroupe(groupe, avant string) (int, string) {
	s.t.Helper()
	return s.postForm("/groupes/"+url.PathEscape(groupe)+"/ordre", url.Values{"avant": {avant}})
}

// quatreGroupes crée A, B, C et D, dans cet ordre.
func quatreGroupes(svc *service) {
	for _, g := range []string{"A", "B", "C", "D"} {
		svc.creerGroupe(g)
	}
}

func TestGroupeMovesUpAndDownAndSansGroupeStaysLast(t *testing.T) {
	svc, _ := troisStores(t, nil)
	quatreGroupes(svc)

	svc.deplacerGroupe("C", "haut")
	svc.deplacerGroupe("A", "haut") // déjà premier
	svc.deplacerGroupe("D", "bas")  // déjà dernier, avant « Sans groupe »
	code, body := svc.deplacerGroupe("A", "bas")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	if got, want := noms(groupes(t, body)), []string{"C", "A", "B", "D", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
}

func TestGroupeIsPlacedBeforeAnotherGroupeOrLast(t *testing.T) {
	svc, _ := troisStores(t, nil)
	quatreGroupes(svc)

	svc.placerGroupe("D", "B")
	code, body := svc.placerGroupe("A", sansGroupe)

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	if got, want := noms(groupes(t, body)), []string{"D", "B", "C", "A", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
}

func TestSansGroupeAndUnknownGroupesDoNotMove(t *testing.T) {
	svc, _ := troisStores(t, nil)
	quatreGroupes(svc)

	for _, c := range []struct {
		name string
		send func() (int, string)
		code int
	}{
		{"Sans groupe ↑", func() (int, string) { return svc.deplacerGroupe(sansGroupe, "haut") }, http.StatusBadRequest},
		{"Sans groupe avant A", func() (int, string) { return svc.placerGroupe(sansGroupe, "A") }, http.StatusBadRequest},
		{"inconnu ↑", func() (int, string) { return svc.deplacerGroupe("Inconnu", "haut") }, http.StatusNotFound},
		{"avant un inconnu", func() (int, string) { return svc.placerGroupe("D", "Inconnu") }, http.StatusNotFound},
		{"sens inconnu", func() (int, string) { return svc.deplacerGroupe("D", "gauche") }, http.StatusBadRequest},
	} {
		code, body := c.send()

		if code != c.code {
			t.Errorf("%s : code %d, attendu %d", c.name, code, c.code)
		}
		assertRangementErreur(t, body)
	}
	_, body := svc.get("/")
	if got, want := noms(groupes(t, body)), []string{"A", "B", "C", "D", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
}

// renommerGroupe envoie le formulaire « Renommer » du menu ⋯ d'un Groupe.
func (s *service) renommerGroupe(groupe, nom string) (int, string) {
	s.t.Helper()
	return s.postForm("/groupes/"+url.PathEscape(groupe)+"/nom", url.Values{"nom": {nom}})
}

func TestRenamedGroupeKeepsItsPlaceAndItsStores(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")
	svc.creerGroupe("Hors site")
	svc.ranger("zeta", "Unraid")
	svc.ranger("arr", "Unraid")

	code, body := svc.renommerGroupe("Unraid", "  NAS ")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	gs := groupes(t, body)
	if got, want := noms(gs), []string{"NAS", "Hors site", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
	if got, want := gs[0].stores, []string{"zeta", "arr"}; !slices.Equal(got, want) {
		t.Errorf("NAS = %q, attendu %q", got, want)
	}
	// Le Groupe se range désormais sous son nouveau nom.
	if code, _ := svc.ranger("immich", "NAS"); code != http.StatusOK {
		t.Errorf("ranger dans NAS : code %d", code)
	}
}

func TestRenamingAGroupeToItsOwnNameChangesNothing(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")

	code, body := svc.renommerGroupe("Unraid", " Unraid ")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	if got, want := noms(groupes(t, body)), []string{"Unraid", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
}

func TestRefusedRenamesChangeNothingAndSayWhy(t *testing.T) {
	for _, c := range []struct {
		name, groupe, nom string
		code              int
	}{
		{"vide", "Unraid", "", http.StatusBadRequest},
		{"blancs", "Unraid", "   ", http.StatusBadRequest},
		{"en double", "Unraid", " Hors site", http.StatusBadRequest},
		{"réservé", "Unraid", sansGroupe, http.StatusBadRequest},
		{"Sans groupe", sansGroupe, "Autre", http.StatusBadRequest},
		{"inconnu", "Inconnu", "Autre", http.StatusNotFound},
	} {
		t.Run(c.name, func(t *testing.T) {
			svc, _ := troisStores(t, nil)
			svc.creerGroupe("Unraid")
			svc.creerGroupe("Hors site")

			code, body := svc.renommerGroupe(c.groupe, c.nom)

			if code != c.code {
				t.Errorf("code %d, attendu %d", code, c.code)
			}
			assertRangementErreur(t, body)
			_, body = svc.get("/")
			if got, want := noms(groupes(t, body)), []string{"Unraid", "Hors site", "Sans groupe"}; !slices.Equal(got, want) {
				t.Errorf("Groupes %q, attendu %q", got, want)
			}
		})
	}
}

// supprimerGroupe confirme « Supprimer » dans le menu ⋯ d'un Groupe.
func (s *service) supprimerGroupe(groupe string) (int, string) {
	s.t.Helper()
	return s.postForm("/groupes/"+url.PathEscape(groupe)+"/suppression", nil)
}

func TestDeletedGroupeSendsItsStoresToTheEndOfSansGroupeInOrder(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")
	svc.creerGroupe("Hors site")
	svc.ranger("zeta", "Unraid")
	svc.ranger("arr", "Unraid")

	code, body := svc.supprimerGroupe("Unraid")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	gs := groupes(t, body)
	if got, want := noms(gs), []string{"Hors site", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
	if got, want := rangement(gs)["Sans groupe"], []string{"immich", "zeta", "arr"}; !slices.Equal(got, want) {
		t.Errorf("Sans groupe = %q, attendu %q", got, want)
	}
	// Le nom est de nouveau libre.
	if code, _ := svc.creerGroupe("Unraid"); code != http.StatusOK {
		t.Errorf("recréer Unraid : code %d", code)
	}
}

func TestSansGroupeAndUnknownGroupesCannotBeDeleted(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")
	svc.ranger("arr", "Unraid")

	for groupe, want := range map[string]int{sansGroupe: http.StatusBadRequest, "Inconnu": http.StatusNotFound} {
		code, body := svc.supprimerGroupe(groupe)

		if code != want {
			t.Errorf("%s : code %d, attendu %d", groupe, code, want)
		}
		assertRangementErreur(t, body)
	}
	_, body := svc.get("/")
	want := map[string][]string{"Unraid": {"arr"}, "Sans groupe": {"immich", "zeta"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
}

func TestOrderOfGroupesAndOfStoresSurvivesARestart(t *testing.T) {
	state := map[string]string{"PLAKAR_UI_STATE": filepath.Join(t.TempDir(), "state.json")}
	svc, _ := troisStores(t, state)
	svc.creerGroupe("Unraid")
	svc.creerGroupe("Hors site")
	svc.renommerGroupe("Hors site", "Cloud")
	svc.placerGroupe("Cloud", "Unraid")
	svc.ranger("arr", "Unraid")
	svc.ranger("zeta", "Unraid")
	svc.deplacerStore("zeta", "haut")
	// Un Groupe supprimé ne revient pas au redémarrage.
	svc.creerGroupe("Jetable")
	svc.supprimerGroupe("Jetable")
	svc.stop()

	svc, _ = troisStores(t, state)
	_, body := svc.get("/")

	gs := groupes(t, body)
	if got, want := noms(gs), []string{"Cloud", "Unraid", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
	want := map[string][]string{"Cloud": nil, "Unraid": {"zeta", "arr"}, "Sans groupe": {"immich"}}
	if got := rangement(gs); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
}

func TestOrderOfSansGroupeSurvivesARestart(t *testing.T) {
	state := map[string]string{"PLAKAR_UI_STATE": filepath.Join(t.TempDir(), "state.json")}
	svc, _ := troisStores(t, state)
	svc.deplacerStore("zeta", "haut")
	svc.deplacerStore("zeta", "haut")
	svc.stop()

	svc, _ = troisStores(t, state)
	_, body := svc.get("/")

	if got, want := rangement(groupes(t, body))["Sans groupe"], []string{"zeta", "arr", "immich"}; !slices.Equal(got, want) {
		t.Errorf("Sans groupe = %q, attendu %q", got, want)
	}
}

func TestOrderRenameAndDeleteAreForbiddenToAnUnknownIdentity(t *testing.T) {
	svc, _ := troisStores(t, nil)
	quatreGroupes(svc)
	svc.ranger("arr", "A")

	for _, path := range []string{
		"/stores/zeta/ordre?sens=haut",
		"/stores/zeta/groupe?groupe=A&avant=arr",
		"/groupes/D/ordre?sens=haut",
		"/groupes/D/ordre?avant=A",
		"/groupes/A/nom?nom=Intrus",
		"/groupes/A/suppression",
	} {
		req := svc.request(http.MethodPost, path)
		if code, _ := svc.do(req, "intrus@example.com"); code != http.StatusForbidden {
			t.Errorf("POST %s : code %d, attendu %d", path, code, http.StatusForbidden)
		}
	}

	_, body := svc.get("/")
	gs := groupes(t, body)
	if got, want := noms(gs), []string{"A", "B", "C", "D", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
	if got, want := rangement(gs)["Sans groupe"], []string{"immich", "zeta"}; !slices.Equal(got, want) {
		t.Errorf("Sans groupe = %q, attendu %q", got, want)
	}
}

// actions renvoie, pour chaque bouton d'un formulaire sous n, l'action du
// formulaire, la valeur du bouton s'il en a une, et s'il est désactivé
// (« /groupes/A/ordre haut disabled »).
func actions(n *html.Node) []string {
	var out []string
	var walk func(n *html.Node, action string)
	walk = func(n *html.Node, action string) {
		if n.Type == html.ElementNode {
			if n.Data == "form" {
				action, _ = attr(n, "action")
			}
			if action != "" && n.Data == "button" {
				line := action
				if v, ok := attr(n, "value"); ok {
					line += " " + v
				}
				if _, ok := attr(n, "disabled"); ok {
					line += " disabled"
				}
				out = append(out, line)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, action)
		}
	}
	walk(n, "")
	return out
}

// entete renvoie l'en-tête du Groupe de clé cle.
func entete(t *testing.T, body, cle string) *html.Node {
	t.Helper()
	g := findElement(t, body, func(n *html.Node) bool { v, _ := attr(n, "data-groupe"); return v == cle })
	if g == nil {
		t.Fatalf("Groupe %q absent de la page", cle)
	}
	h := findIn(g, func(n *html.Node) bool { _, ok := attr(n, "data-groupe-entete"); return ok })
	if h == nil {
		t.Fatalf("Groupe %q sans en-tête", cle)
	}
	return h
}

func TestGroupeHeadersOfferOrderRenameAndDeleteExceptSansGroupe(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("A")
	svc.creerGroupe("B/C")

	_, body := svc.get("/")

	for cle, want := range map[string][]string{
		"A":   {"/groupes/A/ordre haut disabled", "/groupes/A/ordre bas", "/groupes/A/nom", "/groupes/A/suppression"},
		"B/C": {"/groupes/B%2FC/ordre haut", "/groupes/B%2FC/ordre bas disabled", "/groupes/B%2FC/nom", "/groupes/B%2FC/suppression"},
	} {
		if got := actions(entete(t, body, cle)); !slices.Equal(got, want) {
			t.Errorf("en-tête de %s : %q, attendu %q", cle, got, want)
		}
	}
	if got := actions(entete(t, body, sansGroupe)); len(got) != 0 {
		t.Errorf("en-tête de Sans groupe : %q, attendu aucune action", got)
	}
}

func TestStoreDepliageOffersUpAndDownWithinItsGroupe(t *testing.T) {
	svc, _ := troisStores(t, nil)

	_, body := svc.get("/")

	for store, want := range map[string][]string{
		"arr":    {"/stores/arr/ordre haut disabled", "/stores/arr/ordre bas"},
		"immich": {"/stores/immich/ordre haut", "/stores/immich/ordre bas"},
		"zeta":   {"/stores/zeta/ordre haut", "/stores/zeta/ordre bas disabled"},
	} {
		el := findElement(t, body, func(n *html.Node) bool { v, _ := attr(n, "data-store"); return v == store })
		var got []string
		for _, a := range actions(el) {
			if strings.Contains(a, "/ordre") {
				got = append(got, a)
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s : %q, attendu %q", store, got, want)
		}
	}
}

func TestARefusedRenameReopensTheMenuWithTheRefusedName(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")
	svc.creerGroupe("Hors site")

	_, body := svc.renommerGroupe("Unraid", "Hors site")

	menu := findIn(entete(t, body, "Unraid"), func(n *html.Node) bool { _, ok := attr(n, "data-groupe-menu"); return ok })
	if _, open := attr(menu, "open"); !open {
		t.Errorf("le menu ⋯ de Unraid reste fermé")
	}
	input := findIn(menu, func(n *html.Node) bool { v, _ := attr(n, "name"); return n.Data == "input" && v == "nom" })
	if v, _ := attr(input, "value"); v != "Hors site" {
		t.Errorf("champ du nom = %q, attendu « Hors site »", v)
	}
	if _, open := attr(findIn(entete(t, body, "Hors site"), func(n *html.Node) bool { _, ok := attr(n, "data-groupe-menu"); return ok }), "open"); open {
		t.Errorf("le menu ⋯ de Hors site s'est ouvert")
	}
}

func TestPlacingAStoreBeforeItselfOutsideTheTargetGroupeIsRefused(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")

	code, body := svc.placer("zeta", "Unraid", "zeta")

	if code != http.StatusNotFound {
		t.Errorf("code %d, attendu %d", code, http.StatusNotFound)
	}
	assertRangementErreur(t, body)
}

func TestStoreIsPlacedBeforeAnotherInSansGroupe(t *testing.T) {
	svc, _ := troisStores(t, nil)

	_, body := svc.placer("zeta", sansGroupe, "arr")

	if got, want := rangement(groupes(t, body))["Sans groupe"], []string{"zeta", "arr", "immich"}; !slices.Equal(got, want) {
		t.Errorf("Sans groupe = %q, attendu %q", got, want)
	}
}

func TestGroupePlacedBeforeAnEmptyPositionGoesLast(t *testing.T) {
	svc, _ := troisStores(t, nil)
	quatreGroupes(svc)

	_, body := svc.placerGroupe("A", "")

	if got, want := noms(groupes(t, body)), []string{"B", "C", "D", "A", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
}

func TestAGroupeWithASlashInItsNameIsRenamedMovedAndDeleted(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("A")
	svc.creerGroupe("B/C")
	svc.ranger("arr", "B/C")

	svc.deplacerGroupe("B/C", "haut")
	_, body := svc.renommerGroupe("B/C", "D/E")
	if got, want := noms(groupes(t, body)), []string{"D/E", "A", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
	_, body = svc.supprimerGroupe("D/E")
	if got, want := noms(groupes(t, body)), []string{"A", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
}

func TestAFailedMoveReopensTheStoreDepliage(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")
	svc.ranger("arr", "Unraid")

	_, body := svc.placer("zeta", "Unraid", "immich")

	el := findElement(t, body, func(n *html.Node) bool { v, _ := attr(n, "data-store"); return v == "zeta" })
	if _, open := attr(el, "open"); !open {
		t.Errorf("le dépliage de zeta reste fermé")
	}
	assertRangementErreur(t, body)
}
