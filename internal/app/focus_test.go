package app_test

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

// Focus sur un Groupe : un clic sur le nom d'un Groupe n'affiche plus que ce
// Groupe, à une adresse qu'on peut mettre en favori.

// focus ouvre le focus de ce Groupe (ou de sansGroupe).
func (s *service) focus(groupe string) (int, string) {
	s.t.Helper()
	return s.get("/?groupe=" + url.QueryEscape(groupe))
}

func TestFocusShowsOnlyThatGroupeWithALinkBackToAllGroupes(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Médias")
	svc.creerGroupe("Docker")
	svc.ranger("immich", "Médias")
	svc.ranger("arr", "Docker")

	code, body := svc.focus("Médias")

	if code != http.StatusOK {
		t.Fatalf("code %d, attendu %d\n%s", code, http.StatusOK, body)
	}
	gs := groupes(t, body)
	if len(gs) != 1 || gs[0].nom != "Médias" || !slices.Equal(gs[0].stores, []string{"immich"}) {
		t.Errorf("focus de Médias = %v, attendu le seul Groupe Médias avec immich", gs)
	}
	assertLienTousLesGroupes(t, body)
}

// assertLienTousLesGroupes vérifie que la page porte le lien « ← Tous les
// groupes » vers la vue d'ensemble.
func assertLienTousLesGroupes(t *testing.T, body string) {
	t.Helper()
	a := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-tous-les-groupes"); return ok })
	if a == nil {
		t.Fatalf("aucun lien « ← Tous les groupes » :\n%s", body)
	}
	if href, _ := attr(a, "href"); a.Data != "a" || href != "/" {
		t.Errorf("lien « ← Tous les groupes » : <%s href=%q>, attendu <a href=\"/\">", a.Data, href)
	}
	if got := text(a); !strings.Contains(got, "Tous les groupes") {
		t.Errorf("lien = %q, doit dire « ← Tous les groupes »", got)
	}
}

func TestFocusOfSansGroupeShowsTheStoresLeftToRange(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Médias")
	svc.ranger("immich", "Médias")

	code, body := svc.focus(sansGroupe)

	if code != http.StatusOK {
		t.Fatalf("code %d, attendu %d\n%s", code, http.StatusOK, body)
	}
	gs := groupes(t, body)
	if len(gs) != 1 || gs[0].cle != sansGroupe || !slices.Equal(gs[0].stores, []string{"arr", "zeta"}) {
		t.Errorf("focus de « Sans groupe » = %v, attendu le seul « Sans groupe » avec arr et zeta", gs)
	}
	assertLienTousLesGroupes(t, body)
}

func TestGroupeNameInItsHeaderOpensItsFocus(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Médias & co")
	svc.ranger("immich", "Médias & co")

	_, body := svc.get("/")

	for cle, nom := range map[string]string{"Médias & co": "Médias & co", sansGroupe: "Sans groupe"} {
		h := findIn(entete(t, body, cle), func(n *html.Node) bool { _, ok := attr(n, "data-groupe-nom"); return ok })
		a := findIn(h, func(n *html.Node) bool { return n.Data == "a" })
		if a == nil {
			t.Errorf("le nom de %q ne mène nulle part", nom)
			continue
		}
		href, _ := attr(a, "href")
		_, focused := svc.get(href)
		gs := groupes(t, focused)
		if len(gs) != 1 || gs[0].nom != nom {
			t.Errorf("le nom de %q mène à %q, qui affiche %v ; attendu son seul focus", nom, href, gs)
		}
	}
}

func TestFocusOfAnUnknownGroupeSaysSoAndLinksBackToAllGroupes(t *testing.T) {
	svc, _ := troisStores(t, nil)
	svc.creerGroupe("Médias")
	svc.renommerGroupe("Médias", "Vidéos")

	code, body := svc.focus("Médias")

	if code != http.StatusOK {
		t.Fatalf("code %d, attendu %d\n%s", code, http.StatusOK, body)
	}
	if gs := groupes(t, body); len(gs) != 0 {
		t.Errorf("focus d'un Groupe inconnu = %v, attendu aucun Groupe", gs)
	}
	msg := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-focus-inconnu"); return ok })
	if msg == nil || !strings.Contains(text(msg), "Médias") {
		t.Errorf("aucun message nommant le Groupe inconnu « Médias » :\n%s", body)
	}
	if b := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-tout-relever"); return ok }); b != nil {
		t.Errorf("le focus d'un Groupe inconnu porte un bouton de Relevé %q, attendu aucun", text(b))
	}
	assertLienTousLesGroupes(t, body)
}

// releverGroupe appuie sur le bouton « Relever le groupe » du bandeau en focus.
func (s *service) releverGroupe(groupe string) (int, string) {
	s.t.Helper()
	return s.postForm("/releve", url.Values{"groupe": {groupe}})
}

func TestBandeauInFocusCountsOnlyTheStoresOfTheGroupe(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"recent":{}}` + "\n" + `{"ancien":{}}` + "\n" + `{"panne":{}}` + "\n")
	plakar.ls("recent", response{Stdout: "2026-11-06T11:00:05Z   93ebf1a2   1.3 GiB       21s /root/docker\n"})
	plakar.ls("ancien", response{Stdout: "2026-11-05T14:00:00Z   a4e9d6ad   1.3 GiB       21s /root/docker\n"})
	plakar.ls("panne", response{Stderr: "plakar: could not resolve repository: @panne\n", Code: 1})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()
	svc.creerGroupe("Docker")
	svc.ranger("recent", "Docker")
	svc.ranger("ancien", "Docker")

	_, body := svc.focus("Docker")
	assertCompteurs(t, body, map[string]string{"a-jour": "1", "en-retard": "1", "injoignable": "0"})

	_, body = svc.focus(sansGroupe)
	assertCompteurs(t, body, map[string]string{"a-jour": "0", "en-retard": "0", "injoignable": "1"})
}

func TestReleverLeGroupeRelevesOnlyItsStoresAndSkipsThoseInProgress(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n" + `{"lent":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.ls("immich", response{Stdout: fixture(t, "ls-arr.txt")})
	// Le premier Relevé de lent n'est pas terminé pendant le test.
	plakar.ls("lent", response{DelayMS: 60_000})
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, nil)
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:00" })
	svc.waitRow("immich", func(r row) bool { return r["releve"] == "2026-11-06 18:00" })
	svc.waitRow("lent", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })
	svc.creerGroupe("Docker")
	svc.ranger("arr", "Docker")
	svc.ranger("lent", "Docker")
	_, body := svc.focus("Docker")
	assertReleverLeGroupe(t, body, "Docker")
	clk.advance(5 * time.Minute)

	code, body := svc.releverGroupe("Docker")

	if code != http.StatusOK {
		t.Fatalf("code %d après Relever le groupe\n%s", code, body)
	}
	if gs := groupes(t, body); len(gs) != 1 || gs[0].nom != "Docker" {
		t.Errorf("après Relever le groupe, la page affiche %v ; attendu le focus de Docker", gs)
	}
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:05" })
	if got := lsCalls(plakar, "lent"); got != 1 {
		t.Errorf("%d Relevés de lent, attendu 1 : Relever le groupe en a lancé un second", got)
	}
	if got := lsCalls(plakar, "immich"); got != 1 {
		t.Errorf("%d Relevés de immich, attendu 1 : il n'est pas dans Docker", got)
	}
}

func TestReleverAnUnknownGroupeRelevesNothingAndSaysWhy(t *testing.T) {
	svc, plakar := troisStores(t, nil)

	code, body := svc.releverGroupe("Fantôme")

	if code != http.StatusNotFound {
		t.Errorf("code %d, attendu %d\n%s", code, http.StatusNotFound, body)
	}
	assertRangementErreur(t, body)
	for _, s := range []string{"zeta", "arr", "immich"} {
		if got := lsCalls(plakar, s); got != 1 {
			t.Errorf("%d Relevés de %s, attendu 1", got, s)
		}
	}
}

func TestFocusRefreshesItselfOnlyWhileOneOfItsStoresIsBeingReleved(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"lent":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.ls("lent", response{DelayMS: 60_000})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:00" })
	svc.waitRow("lent", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })
	svc.creerGroupe("Docker")
	svc.ranger("arr", "Docker")

	_, body := svc.focus("Docker")
	if got, ok := refresh(t, body); ok {
		t.Errorf("rafraîchissement %q dans le focus de Docker, dont aucun Store n'est en cours de Relevé", got)
	}
	_, body = svc.focus(sansGroupe)
	if _, ok := refresh(t, body); !ok {
		t.Errorf("aucun rafraîchissement dans le focus de « Sans groupe », où lent est en cours de Relevé")
	}
}

// assertReleverLeGroupe vérifie que le bandeau du focus porte le bouton
// « Relever le groupe », qui vise ce Groupe.
func assertReleverLeGroupe(t *testing.T, body, groupe string) {
	t.Helper()
	bandeau := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-bandeau"); return ok })
	if bandeau == nil {
		t.Fatalf("aucun bandeau :\n%s", body)
	}
	button := findIn(bandeau, func(n *html.Node) bool { _, ok := attr(n, "data-tout-relever"); return ok })
	if button == nil {
		t.Fatalf("aucun bouton de Relevé dans le bandeau :\n%s", body)
	}
	if got := text(button); !strings.Contains(got, "Relever le groupe") {
		t.Errorf("bouton = %q en focus, doit dire « Relever le groupe »", got)
	}
	input := findIn(bandeau, func(n *html.Node) bool { v, _ := attr(n, "name"); return n.Data == "input" && v == "groupe" })
	if input == nil {
		t.Fatalf("le bouton « Relever le groupe » ne porte aucun Groupe :\n%s", body)
	}
	if v, _ := attr(input, "value"); v != groupe {
		t.Errorf("le bouton « Relever le groupe » ne vise pas %q :\n%s", groupe, body)
	}
}
