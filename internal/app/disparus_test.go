package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// Stores disparus : un Store retiré de la config plakar alors qu'il était
// rangé dans un Groupe garde sa place, jusqu'à ce que Jobierre fasse le ménage
// ou qu'un Store du même nom réapparaisse.

// disparuVu est un Store disparu tel que la page l'affiche.
type disparuVu struct {
	mention string
	// retirer : la page propose « Retirer du groupe ».
	retirer bool
}

// disparus lit les Stores disparus de la page (<… data-store="…"
// data-disparu>), par nom.
func disparus(t *testing.T, body string) map[string]disparuVu {
	t.Helper()
	out := map[string]disparuVu{}
	var walk func(n *html.Node, store string)
	walk = func(n *html.Node, store string) {
		if n.Type == html.ElementNode {
			if name, ok := attr(n, "data-store"); ok {
				store = ""
				if _, ok := attr(n, "data-disparu"); ok {
					store = name
					out[store] = disparuVu{}
				}
			}
			if store != "" {
				d := out[store]
				if _, ok := attr(n, "data-disparu-mention"); ok {
					d.mention = text(n)
				}
				if action, ok := attr(n, "action"); ok && action == "/stores/"+url.PathEscape(store)+"/retrait" {
					d.retirer = true
				}
				out[store] = d
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, store)
		}
	}
	walk(parseHTML(t, body), "")
	return out
}

// retirerDuGroupe appuie sur « Retirer du groupe » d'un Store disparu.
func (s *service) retirerDuGroupe(store string) (int, string) {
	s.t.Helper()
	return s.postForm("/stores/"+url.PathEscape(store)+"/retrait", nil)
}

// rmEtAdd fait retirer le Store de la config par `store rm <store>`, puis
// l'y remettre par `store add <store> location=…`, avec ces Stores restants
// et ces Stores après l'ajout (chacun une ligne de `store show -json`).
func rmEtAdd(plakar *fakePlakar, store string, apresRm, apresAdd []string) {
	jsonl := func(names []string) string {
		out := ""
		for _, n := range names {
			out += `{"` + n + `":{}}` + "\n"
		}
		return out
	}
	plakar.on([]string{"store", "rm", store}, response{
		Replace: []response{{Args: storeShowArgs, Stdout: jsonl(apresRm)}},
	})
	plakar.on([]string{"store", "add", store, "location=http://unraid:9880"}, response{
		Replace: []response{{Args: storeShowArgs, Stdout: jsonl(apresAdd)}},
	})
}

// troisStoresRanges démarre le service sur zeta, arr et immich, arr et zeta
// rangés dans « Unraid » (dans cet ordre), immich « Sans groupe ». Le faux
// plakar sait retirer puis remettre arr, et immich, et ajouter neuf.
func troisStoresRanges(t *testing.T, env map[string]string) (*service, *fakePlakar) {
	t.Helper()
	plakar := newFakePlakar(t)
	plakar.stores(`{"zeta":{}}` + "\n" + `{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	for _, s := range []string{"zeta", "arr", "immich", "neuf"} {
		plakar.ls(s, response{Stdout: fixture(t, "ls-arr.txt")})
	}
	rmEtAdd(plakar, "arr", []string{"zeta", "immich"}, []string{"zeta", "arr", "immich"})
	rmEtAdd(plakar, "immich", []string{"zeta", "arr"}, []string{"zeta", "arr", "immich"})
	// neuf vient prendre le port d'arr, une fois arr retiré.
	plakar.on([]string{"store", "add", "neuf", "location=http://unraid:9880"}, response{
		Replace: []response{{Args: storeShowArgs, Stdout: `{"zeta":{}}` + "\n" + `{"immich":{}}` + "\n" + `{"neuf":{}}` + "\n"}},
	})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), env)
	svc.dashboard()
	svc.creerGroupe("Unraid")
	svc.ranger("arr", "Unraid")
	svc.ranger("zeta", "Unraid")
	return svc, plakar
}

func TestARangedStoreRemovedFromTheConfigStaysInItsPlaceAsStoreDisparu(t *testing.T) {
	svc, _ := troisStoresRanges(t, nil)

	code, body := svc.commande("store rm arr")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	want := map[string][]string{"Unraid": {"arr", "zeta"}, "Sans groupe": {"immich"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
	d, ok := disparus(t, body)["arr"]
	if !ok {
		t.Fatalf("arr n'est pas affiché comme Store disparu :\n%s", body)
	}
	if d.mention != "Store supprimé" || !d.retirer {
		t.Errorf("Store disparu %+v, attendu « Store supprimé » et « Retirer du groupe »", d)
	}
	// Ni État, ni Relevé, ni valeurs : rien de périmé, peut-être d'un autre
	// Kloset.
	_, rows, _ := parseDashboard(t, body)
	if len(rows["arr"]) != 0 {
		t.Errorf("le Store disparu affiche des valeurs : %v", rows["arr"])
	}
}

func TestAStoreSansGroupeRemovedFromTheConfigIsSimplyForgotten(t *testing.T) {
	svc, _ := troisStoresRanges(t, nil)

	_, body := svc.commande("store rm immich")

	if names, _, _ := parseDashboard(t, body); slices.Contains(names, "immich") {
		t.Errorf("immich, « Sans groupe », est encore affiché : %q", names)
	}
}

func TestAStoreDisparuFreesItsExplorationPort(t *testing.T) {
	// Trois ports pour trois Stores : neuf n'en a un que si arr rend le sien.
	base := freePorts(3)
	svc, _ := troisStoresRanges(t, map[string]string{"PLAKAR_UI_PORTS": fmt.Sprintf("%d-%d", base, base+2)})
	svc.commande("store rm arr")

	_, body := svc.commande("plakar store add neuf location=http://unraid:9880")

	_, rows, _ := parseDashboard(t, body)
	if msg, ok := rows["neuf"]["port-message"]; ok {
		t.Errorf("neuf n'a pas de port : %q", msg)
	}
}

func TestAStoreDisparuBackInTheConfigTakesBackItsPlaceAndSeuilWithANewReleve(t *testing.T) {
	svc, plakar := troisStoresRanges(t, nil)
	svc.setSeuil("arr", "8d")
	svc.commande("store rm arr")

	_, body := svc.commande("plakar store add arr location=http://unraid:9880")

	want := map[string][]string{"Unraid": {"arr", "zeta"}, "Sans groupe": {"immich"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
	if _, ok := disparus(t, body)["arr"]; ok {
		t.Errorf("arr est encore un Store disparu")
	}
	got := svc.waitRow("arr", func(r row) bool { return r["releve"] != "" })
	if got["seuil"] != "8d" {
		t.Errorf("Seuil de arr %q, attendu 8d", got["seuil"])
	}
	if _, ok := got["port-message"]; ok {
		t.Errorf("arr n'a pas de port : %q", got["port-message"])
	}
	if n := lsCalls(plakar, "arr"); n != 2 {
		t.Errorf("%d Relevés de arr, attendu 2 : un au démarrage, un au retour", n)
	}
}

func TestRetirerDuGroupeForgetsTheStoreDisparuAndItsSeuil(t *testing.T) {
	svc, _ := troisStoresRanges(t, nil)
	svc.setSeuil("arr", "8d")
	svc.commande("store rm arr")

	code, body := svc.retirerDuGroupe("arr")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	want := map[string][]string{"Unraid": {"zeta"}, "Sans groupe": {"immich"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
	// Revenu dans la config, arr est un Store nouveau : en fin de « Sans
	// groupe », au Seuil par défaut.
	_, body = svc.commande("plakar store add arr location=http://unraid:9880")
	want = map[string][]string{"Unraid": {"zeta"}, "Sans groupe": {"immich", "arr"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
	if got := svc.waitRow("arr", func(r row) bool { return true }); got["seuil"] == "8d" {
		t.Errorf("arr a gardé son Seuil 8d")
	}
}

func TestRetirerDuGroupeRefusesAStoreThatIsNotDisparu(t *testing.T) {
	svc, _ := troisStoresRanges(t, nil)

	for store, wantCode := range map[string]int{"arr": http.StatusBadRequest, "inconnu": http.StatusNotFound} {
		code, body := svc.retirerDuGroupe(store)

		if code != wantCode {
			t.Errorf("%s : code %d, attendu %d", store, code, wantCode)
		}
		want := map[string][]string{"Unraid": {"arr", "zeta"}, "Sans groupe": {"immich"}}
		if got := rangement(groupes(t, body)); !equalRangement(got, want) {
			t.Errorf("%s : rangement %q, attendu %q", store, got, want)
		}
	}
}

// compteurDisparus lit le compteur 👻 du bandeau, vide s'il n'y en a pas.
func compteurDisparus(t *testing.T, body string) string {
	t.Helper()
	el := findElement(t, body, func(n *html.Node) bool { v, _ := attr(n, "data-compteur"); return v == "disparu" })
	if el == nil {
		return ""
	}
	return text(el)
}

func TestBandeauCountsStoresDisparusApartFromTheEtats(t *testing.T) {
	svc, _ := troisStoresRanges(t, nil)
	svc.creerGroupe("Hors site")
	svc.ranger("immich", "Hors site")
	if _, body := svc.get("/"); compteurDisparus(t, body) != "" {
		t.Errorf("compteur 👻 affiché sans Store disparu")
	}
	svc.commande("store rm arr")

	_, body := svc.get("/")

	if got := compteurDisparus(t, body); got != "1" {
		t.Errorf("compteur 👻 %q, attendu 1", got)
	}
	// Un Store disparu n'a pas d'État : il n'est compté dans aucun.
	assertCompteurs(t, body, map[string]string{"a-jour": "2", "en-retard": "0", "injoignable": "0"})
	// En focus, seuls les Stores disparus du Groupe comptent.
	if _, body := svc.focus("Unraid"); compteurDisparus(t, body) != "1" {
		t.Errorf("focus Unraid : compteur 👻 %q, attendu 1", compteurDisparus(t, body))
	}
	if _, body := svc.focus("Hors site"); compteurDisparus(t, body) != "" {
		t.Errorf("focus Hors site : compteur 👻 %q, attendu aucun", compteurDisparus(t, body))
	}
}

func TestToutReleverAndReleverLeGroupeIgnoreStoresDisparus(t *testing.T) {
	svc, plakar := troisStoresRanges(t, nil)
	svc.commande("store rm arr")

	for _, relever := range []func() (int, string){svc.toutRelever, func() (int, string) { return svc.releverGroupe("Unraid") }} {
		if code, body := relever(); code != http.StatusOK {
			t.Fatalf("code %d\n%s", code, body)
		}
		svc.waitRow("zeta", func(r row) bool { return r["releve-en-cours"] == "" })
	}

	if n := lsCalls(plakar, "arr"); n != 1 {
		t.Errorf("%d Relevés de arr, attendu 1 : le Store disparu a été relevé", n)
	}
	if n := lsCalls(plakar, "zeta"); n != 3 {
		t.Errorf("%d Relevés de zeta, attendu 3", n)
	}
	if strings.Contains(svc.log.String(), "level=ERROR") {
		t.Errorf("erreur journalisée :\n%s", svc.log)
	}
}

func TestSupprimerLeGroupeForgetsItsStoresDisparus(t *testing.T) {
	svc, _ := troisStoresRanges(t, nil)
	svc.setSeuil("arr", "8d")
	svc.commande("store rm arr")

	code, body := svc.supprimerGroupe("Unraid")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	want := map[string][]string{"Sans groupe": {"immich", "zeta"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
	if got := compteurDisparus(t, body); got != "" {
		t.Errorf("compteur 👻 %q, attendu aucun", got)
	}
	// Revenu, arr est un Store nouveau, au Seuil par défaut.
	svc.commande("plakar store add arr location=http://unraid:9880")
	if got := svc.waitRow("arr", func(r row) bool { return true }); got["seuil"] == "8d" {
		t.Errorf("arr a gardé son Seuil 8d")
	}
}

func TestStoresDisparusSurviveARestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	state := map[string]string{"PLAKAR_UI_STATE": statePath}
	svc, _ := troisStoresRanges(t, state)
	svc.setSeuil("arr", "8d")
	svc.commande("store rm arr")
	svc.stop()

	// Au redémarrage, la config plakar n'a toujours pas arr.
	plakar := newFakePlakar(t)
	plakar.stores(`{"zeta":{}}` + "\n" + `{"immich":{}}` + "\n")
	for _, s := range []string{"zeta", "immich", "arr"} {
		plakar.ls(s, response{Stdout: fixture(t, "ls-arr.txt")})
	}
	plakar.on([]string{"store", "add", "arr", "location=http://unraid:9880"}, response{
		Replace: []response{{Args: storeShowArgs, Stdout: `{"zeta":{}}` + "\n" + `{"arr":{}}` + "\n" + `{"immich":{}}` + "\n"}},
	})
	svc = startService(t, plakar, newClock("2026-11-06T17:00:05Z"), state)
	svc.dashboard()
	_, body := svc.get("/")

	want := map[string][]string{"Unraid": {"arr", "zeta"}, "Sans groupe": {"immich"}}
	if got := rangement(groupes(t, body)); !equalRangement(got, want) {
		t.Errorf("rangement %q, attendu %q", got, want)
	}
	if _, ok := disparus(t, body)["arr"]; !ok {
		t.Errorf("arr n'est plus un Store disparu")
	}
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Stores map[string]map[string]any `json:"stores"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	// Du Store disparu, le fichier d'état ne garde que le Seuil.
	if got := f.Stores["arr"]; len(got) != 1 || got["seuil"] != "8d" {
		t.Errorf("état de arr %v, attendu son seul Seuil 8d", got)
	}
	// Son Seuil a survécu : il le reprend à son retour.
	svc.commande("plakar store add arr location=http://unraid:9880")
	if got := svc.waitRow("arr", func(r row) bool { return true }); got["seuil"] != "8d" {
		t.Errorf("Seuil de arr %q, attendu 8d", got["seuil"])
	}
}

func TestRetirerDuGroupeIsForbiddenToAnUnknownIdentity(t *testing.T) {
	svc, _ := troisStoresRanges(t, nil)
	svc.commande("store rm arr")

	req := svc.request(http.MethodPost, "/stores/arr/retrait")
	if code, _ := svc.do(req, "intrus@example.com"); code != http.StatusForbidden {
		t.Errorf("code %d, attendu %d", code, http.StatusForbidden)
	}

	if _, body := svc.get("/"); len(disparus(t, body)) != 1 {
		t.Errorf("l'intrus a retiré arr")
	}
}
