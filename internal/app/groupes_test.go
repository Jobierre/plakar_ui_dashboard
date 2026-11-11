package app_test

import (
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// Groupes : Jobierre range ses Stores dans des Groupes nommés, « Sans groupe »
// en dernier.

// sansGroupe est le mot réservé qui désigne « Sans groupe » dans la page et
// dans les formulaires.
const sansGroupe = "_sans"

// groupeVu est un Groupe tel que la page l'affiche : son nom, ses Stores dans
// l'ordre, le voyant de son en-tête et ses compteurs.
type groupeVu struct {
	cle, nom  string
	stores    []string
	voyant    string
	compteurs map[string]string
}

// groupes lit les Groupes de la page, dans l'ordre d'affichage
// (<… data-groupe="…">).
func groupes(t *testing.T, body string) []groupeVu {
	t.Helper()
	var out []groupeVu
	var walk func(n *html.Node, g *groupeVu)
	walk = func(n *html.Node, g *groupeVu) {
		if n.Type == html.ElementNode {
			if cle, ok := attr(n, "data-groupe"); ok {
				out = append(out, groupeVu{cle: cle, compteurs: map[string]string{}})
				g = &out[len(out)-1]
			}
			if g != nil {
				if _, ok := attr(n, "data-groupe-nom"); ok {
					g.nom = text(n)
				}
				if v, ok := attr(n, "data-groupe-voyant"); ok {
					g.voyant = v
				}
				if v, ok := attr(n, "data-groupe-compteur"); ok {
					g.compteurs[v] = text(n)
				}
				if store, ok := attr(n, "data-store"); ok {
					g.stores = append(g.stores, store)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, g)
		}
	}
	walk(parseHTML(t, body), nil)
	return out
}

// rangement résume les Groupes : le nom de chacun et ses Stores.
func rangement(gs []groupeVu) map[string][]string {
	out := map[string][]string{}
	for _, g := range gs {
		out[g.nom] = g.stores
	}
	return out
}

// noms renvoie le nom de chaque Groupe, dans l'ordre.
func noms(gs []groupeVu) []string {
	out := make([]string, len(gs))
	for i, g := range gs {
		out[i] = g.nom
	}
	return out
}

// creerGroupe envoie le formulaire « + Groupe » du bandeau.
func (s *service) creerGroupe(nom string) (int, string) {
	s.t.Helper()
	return s.postForm("/groupes", url.Values{"nom": {nom}})
}

// ranger envoie la liste déroulante « Groupe » du dépliage d'un Store.
func (s *service) ranger(store, groupe string) (int, string) {
	s.t.Helper()
	return s.postForm("/stores/"+url.PathEscape(store)+"/groupe", url.Values{"groupe": {groupe}})
}

// troisStores démarre le service sur zeta, arr et immich, tous À jour.
func troisStores(t *testing.T, env map[string]string) (*service, *fakePlakar) {
	t.Helper()
	plakar := newFakePlakar(t)
	plakar.stores(`{"zeta":{}}` + "\n" + `{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	for _, s := range []string{"zeta", "arr", "immich"} {
		plakar.ls(s, response{Stdout: fixture(t, "ls-arr.txt")})
	}
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), env)
	svc.dashboard()
	return svc, plakar
}

func TestWithoutGroupesEveryStoreIsSansGroupeInNameOrder(t *testing.T) {
	svc, _ := troisStores(t, nil)

	_, body := svc.get("/")

	gs := groupes(t, body)
	if len(gs) != 1 || gs[0].cle != sansGroupe || gs[0].nom != "Sans groupe" {
		t.Fatalf("Groupes affichés %+v, attendu « Sans groupe » seul", gs)
	}
	if want := []string{"arr", "immich", "zeta"}; !slices.Equal(gs[0].stores, want) {
		t.Errorf("Sans groupe = %q, attendu %q", gs[0].stores, want)
	}
}

func TestAStateFileWithoutGroupesLoadsWithEveryStoreSansGroupe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	// Le fichier d'état de la version précédente : des Stores, aucun Groupe.
	legacy := `{"stores": {"arr": {"seuil": "4h", "port": 0}, "zeta": {}}}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	svc, _ := troisStores(t, map[string]string{"PLAKAR_UI_STATE": path})

	_, body := svc.get("/")

	if got, want := rangement(groupes(t, body)), map[string][]string{"Sans groupe": {"arr", "immich", "zeta"}}; !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"seuil": "4h"})
}

// equalRangement dit si deux rangements ont les mêmes Groupes et les mêmes
// Stores, dans le même ordre.
func equalRangement(a, b map[string][]string) bool {
	return maps.EqualFunc(a, b, slices.Equal)
}

func TestCreatedGroupeAppearsEmptyBeforeSansGroupe(t *testing.T) {
	svc, _ := troisStores(t, nil)

	code, body := svc.creerGroupe("  Unraid ")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	gs := groupes(t, body)
	if got, want := noms(gs), []string{"Unraid", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
	if len(gs[0].stores) != 0 {
		t.Errorf("Unraid contient %q, attendu aucun Store", gs[0].stores)
	}
	svc.creerGroupe("Hors site")
	_, body = svc.get("/")
	if got, want := noms(groupes(t, body)), []string{"Unraid", "Hors site", "Sans groupe"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q : un Groupe créé va après les autres", got, want)
	}
}

func TestRefusedGroupeNamesChangeNothingAndSayWhy(t *testing.T) {
	for name, nom := range map[string]string{
		"vide":      "",
		"blancs":    "   ",
		"en double": " Unraid",
		"réservé":   sansGroupe,
	} {
		t.Run(name, func(t *testing.T) {
			svc, _ := troisStores(t, nil)
			svc.creerGroupe("Unraid")

			code, body := svc.creerGroupe(nom)

			if code != http.StatusBadRequest {
				t.Errorf("code %d, attendu %d", code, http.StatusBadRequest)
			}
			if el := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-rangement-erreur"); return ok }); el == nil || text(el) == "" {
				t.Errorf("aucun message de refus :\n%s", body)
			}
			_, body = svc.get("/")
			if got, want := noms(groupes(t, body)), []string{"Unraid", "Sans groupe"}; !slices.Equal(got, want) {
				t.Errorf("Groupes %q, attendu %q", got, want)
			}
		})
	}
}

func TestRangedStoreGoesToTheEndOfItsNewGroupe(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")

	svc.ranger("zeta", "Unraid")
	svc.ranger("arr", "Unraid")
	code, body := svc.ranger("zeta", sansGroupe)

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	gs := groupes(t, body)
	if got, want := noms(gs), []string{"Unraid", "Sans groupe"}; !slices.Equal(got, want) {
		t.Fatalf("Groupes %q, attendu %q", got, want)
	}
	want := map[string][]string{"Unraid": {"arr"}, "Sans groupe": {"immich", "zeta"}}
	if got := rangement(gs); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
	names, _, _ := parseDashboard(t, body)
	if len(names) != 3 {
		t.Errorf("Stores affichés %q : chacun doit l'être une seule fois", names)
	}
}

func TestSansGroupeIsHiddenOnceEveryStoreIsRanged(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Tout")
	for _, s := range []string{"arr", "immich", "zeta"} {
		svc.ranger(s, "Tout")
	}

	_, body := svc.get("/")

	if got, want := noms(groupes(t, body)), []string{"Tout"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
}

func TestRangingIntoAnUnknownGroupeOrAnUnknownStoreChangesNothing(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")

	for _, c := range []struct{ store, groupe string }{{"arr", "Inconnu"}, {"inconnu", "Unraid"}} {
		code, body := svc.ranger(c.store, c.groupe)

		if code != http.StatusNotFound {
			t.Errorf("%s vers %s : code %d, attendu %d", c.store, c.groupe, code, http.StatusNotFound)
		}
		if el := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-rangement-erreur"); return ok }); el == nil || text(el) == "" {
			t.Errorf("%s vers %s : aucun message de refus", c.store, c.groupe)
		}
	}
	_, body := svc.get("/")
	want := map[string][]string{"Unraid": nil, "Sans groupe": {"arr", "immich", "zeta"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
}

func TestTheGroupeListOfAStoreOffersEveryGroupeAndSelectsItsOwn(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")
	svc.creerGroupe("Hors site")
	svc.ranger("arr", "Hors site")

	_, body := svc.get("/")

	arr := findElement(t, body, func(n *html.Node) bool { v, _ := attr(n, "data-store"); return v == "arr" })
	sel := findIn(arr, func(n *html.Node) bool { v, _ := attr(n, "name"); return n.Data == "select" && v == "groupe" })
	if sel == nil {
		t.Fatalf("aucune liste « Groupe » dans le dépliage de arr")
	}
	var values []string
	selected := ""
	for o := sel.FirstChild; o != nil; o = o.NextSibling {
		if o.Type != html.ElementNode || o.Data != "option" {
			continue
		}
		v, _ := attr(o, "value")
		values = append(values, v)
		if _, ok := attr(o, "selected"); ok {
			selected = v
		}
	}
	if want := []string{"Unraid", "Hors site", sansGroupe}; !slices.Equal(values, want) {
		t.Errorf("choix %q, attendu %q", values, want)
	}
	if selected != "Hors site" {
		t.Errorf("choix retenu %q, attendu « Hors site »", selected)
	}
}

func TestGroupeHeaderShowsTheWorstEtatOfItsStoresAndItsCounters(t *testing.T) {
	plakar := newFakePlakar(t)
	all := []string{"ancien", "coupe", "frais", "coupe2", "neuf", "calme"}
	var jsonl string
	for _, s := range all {
		jsonl += `{"` + s + `":{}}` + "\n"
		if strings.HasPrefix(s, "coupe") {
			plakar.ls(s, response{Stderr: "plakar: connection refused\n", Code: 1})
		} else {
			plakar.ls(s, response{Stdout: fixture(t, "ls-arr.txt")})
		}
	}
	plakar.stores(jsonl)
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()
	// Le Dernier snapshot a 5 h : un Seuil de 1 h met ancien En retard.
	svc.setSeuil("ancien", "1h")
	for _, g := range []struct {
		nom    string
		stores []string
	}{
		{"Mêlé", []string{"ancien", "coupe", "frais"}},
		{"Sans retard", []string{"coupe2", "neuf"}},
		{"Calme", []string{"calme"}},
		{"Vide", nil},
	} {
		svc.creerGroupe(g.nom)
		for _, s := range g.stores {
			svc.ranger(s, g.nom)
		}
	}

	_, body := svc.get("/")

	byName := map[string]groupeVu{}
	for _, g := range groupes(t, body) {
		byName[g.nom] = g
	}
	for nom, want := range map[string]struct {
		voyant    string
		compteurs map[string]string
	}{
		"Mêlé":        {"en-retard", map[string]string{"en-retard": "1", "injoignable": "1", "a-jour": "1"}},
		"Sans retard": {"injoignable", map[string]string{"a-jour": "1", "injoignable": "1"}},
		"Calme":       {"a-jour", map[string]string{"a-jour": "1"}},
		"Vide":        {"", map[string]string{}},
	} {
		g, ok := byName[nom]
		if !ok {
			t.Errorf("Groupe %q absent de la page", nom)
			continue
		}
		if g.voyant != want.voyant {
			t.Errorf("%s : voyant %q, attendu %q", nom, g.voyant, want.voyant)
		}
		if len(g.compteurs) != len(want.compteurs) {
			t.Errorf("%s : compteurs %v, attendu %v", nom, g.compteurs, want.compteurs)
		}
		for v, n := range want.compteurs {
			if g.compteurs[v] != n {
				t.Errorf("%s : compteur %s = %q, attendu %q", nom, v, g.compteurs[v], n)
			}
		}
	}
}

func TestAStoreNewInTheConfigGoesToTheEndOfSansGroupe(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n" + `{"zeta":{}}` + "\n")
	for _, s := range []string{"arr", "immich", "zeta", "aaa"} {
		plakar.ls(s, response{Stdout: fixture(t, "ls-arr.txt")})
	}
	plakar.on([]string{"store", "add", "aaa", "location=http://unraid:9880"}, response{
		Replace: []response{{Args: storeShowArgs, Stdout: `{"aaa":{}}` + "\n" + `{"arr":{}}` + "\n" + `{"immich":{}}` + "\n" + `{"zeta":{}}` + "\n"}},
	})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()
	svc.creerGroupe("Unraid")
	svc.ranger("arr", "Unraid")

	_, body := svc.commande("plakar store add aaa location=http://unraid:9880")

	want := map[string][]string{"Unraid": {"arr"}, "Sans groupe": {"immich", "zeta", "aaa"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
}

func TestRangementSurvivesARestart(t *testing.T) {
	state := map[string]string{"PLAKAR_UI_STATE": filepath.Join(t.TempDir(), "state.json")}
	svc, _ := troisStores(t, state)
	svc.creerGroupe("Unraid")
	svc.creerGroupe("Hors site")
	svc.ranger("zeta", "Unraid")
	svc.ranger("arr", "Unraid")
	svc.ranger("immich", "Hors site")
	svc.stop()

	svc, _ = troisStores(t, state)
	_, body := svc.get("/")

	gs := groupes(t, body)
	if got, want := noms(gs), []string{"Unraid", "Hors site"}; !slices.Equal(got, want) {
		t.Errorf("Groupes %q, attendu %q", got, want)
	}
	want := map[string][]string{"Unraid": {"zeta", "arr"}, "Hors site": {"immich"}}
	if got := rangement(gs); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
}

func TestRangementNeverTouchesThePlakarConfig(t *testing.T) {
	svc, plakar := troisStores(t, nil)

	svc.creerGroupe("Unraid")
	svc.ranger("arr", "Unraid")

	if calls := commandeCalls(plakar); len(calls) != 0 {
		t.Errorf("plakar a été appelé : %q", calls)
	}
}

func TestRangementIsForbiddenToAnUnknownIdentity(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Unraid")

	for _, path := range []string{"/groupes", "/stores/arr/groupe"} {
		req := svc.request(http.MethodPost, path)
		if code, _ := svc.do(req, "intrus@example.com"); code != http.StatusForbidden {
			t.Errorf("POST %s : code %d, attendu %d", path, code, http.StatusForbidden)
		}
	}
	// Même avec un formulaire complet, l'intrus ne range rien.
	req := svc.request(http.MethodPost, "/stores/arr/groupe?groupe=Unraid")
	svc.do(req, "intrus@example.com")
	req = svc.request(http.MethodPost, "/groupes?nom=Intrus")
	svc.do(req, "intrus@example.com")

	_, body := svc.get("/")
	want := map[string][]string{"Unraid": nil, "Sans groupe": {"arr", "immich", "zeta"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
}
