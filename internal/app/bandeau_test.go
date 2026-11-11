package app_test

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestBandeauCountsStoresByEtat(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"recent":{}}` + "\n" + `{"autre":{}}` + "\n" + `{"ancien":{}}` + "\n" + `{"panne":{}}` + "\n")
	plakar.ls("recent", response{Stdout: "2026-11-06T11:00:05Z   93ebf1a2   1.3 GiB       21s /root/docker\n"})
	plakar.ls("autre", response{Stdout: "2026-11-06T12:00:05Z   93ebf1a3   1.3 GiB       21s /root/docker\n"})
	plakar.ls("ancien", response{Stdout: "2026-11-05T14:00:00Z   a4e9d6ad   1.3 GiB       21s /root/docker\n"})
	plakar.ls("panne", response{Stderr: "plakar: could not resolve repository: @panne\n", Code: 1})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()

	_, body := svc.get("/")

	assertCompteurs(t, body, map[string]string{"a-jour": "2", "en-retard": "1", "injoignable": "1"})
}

func TestStoreWithoutFinishedReleveIsCountedNowhere(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"lent":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	// Le premier Relevé de lent n'est pas terminé pendant le test.
	plakar.ls("lent", response{DelayMS: 3000})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.waitRow("arr", func(r row) bool { return r["voyant.etat"] != "" })

	_, body := svc.get("/")

	assertCompteurs(t, body, map[string]string{"a-jour": "1", "en-retard": "0", "injoignable": "0"})
	_, rows, _ := parseDashboard(t, body)
	if got := rows["lent"]["etat"]; got != "Premier Relevé" {
		t.Errorf("État de lent = %q, attendu « Premier Relevé »", got)
	}
	if _, ok := rows["lent"]["voyant"]; ok {
		t.Errorf("un Store sans Relevé terminé ne doit pas avoir de voyant : %v", rows["lent"])
	}
}

func TestEveryVoyantIsAccompaniedByTheWordOfItsEtat(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"recent":{}}` + "\n" + `{"ancien":{}}` + "\n" + `{"panne":{}}` + "\n")
	plakar.ls("recent", response{Stdout: "2026-11-06T11:00:05Z   93ebf1a2   1.3 GiB       21s /root/docker\n"})
	plakar.ls("ancien", response{Stdout: "2026-11-05T14:00:00Z   a4e9d6ad   1.3 GiB       21s /root/docker\n"})
	plakar.ls("panne", response{Stderr: "plakar: could not resolve repository: @panne\n", Code: 1})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	_, rows := svc.dashboard()

	for store, want := range map[string]row{
		"recent": {"voyant.etat": "a-jour", "etat": "À jour"},
		"ancien": {"voyant.etat": "en-retard", "etat": "En retard"},
		"panne":  {"voyant.etat": "injoignable", "etat": "Injoignable"},
	} {
		if _, ok := rows[store]["voyant"]; !ok {
			t.Errorf("%s : aucun voyant", store)
		}
		assertRow(t, rows[store], want)
	}
}

func TestBandeauShowsTheConnectedIdentity(t *testing.T) {
	svc, _ := oneStore(t, nil)

	_, body := svc.get("/")

	bandeau := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-bandeau"); return ok })
	if bandeau == nil {
		t.Fatalf("aucun bandeau :\n%s", body)
	}
	if got := text(bandeau); !strings.Contains(got, testUser) {
		t.Errorf("bandeau = %q, doit montrer l'identité %s", got, testUser)
	}
}

func TestStylesheetIsServedToAnAuthorizedUserOnly(t *testing.T) {
	svc, _ := oneStore(t, nil)
	_, body := svc.get("/")
	link := findElement(t, body, func(n *html.Node) bool {
		rel, _ := attr(n, "rel")
		return n.Data == "link" && rel == "stylesheet"
	})
	if link == nil {
		t.Fatalf("la page ne charge aucune feuille de style :\n%s", body)
	}
	href, _ := attr(link, "href")
	if !strings.HasPrefix(href, "/") {
		t.Fatalf("feuille de style %q : doit être servie par le service lui-même, sans CDN", href)
	}

	req := svc.request(http.MethodGet, href)
	resp := svc.raw(req, testUser)
	if resp.code != http.StatusOK || !strings.HasPrefix(resp.contentType, "text/css") || !strings.Contains(resp.body, "prefers-color-scheme") {
		t.Errorf("GET %s : code %d, type %q, corps de %d octets ; attendu une feuille CSS claire et sombre", href, resp.code, resp.contentType, len(resp.body))
	}

	req = svc.request(http.MethodGet, href)
	if code, _ := svc.do(req, "intrus@example.com"); code != http.StatusForbidden {
		t.Errorf("GET %s par une identité hors liste : code %d, attendu %d", href, code, http.StatusForbidden)
	}
}

// assertCompteurs vérifie les compteurs du bandeau, <… data-compteur="…">.
func assertCompteurs(t *testing.T, body string, want map[string]string) {
	t.Helper()
	for etat, n := range want {
		el := findElement(t, body, func(node *html.Node) bool { v, _ := attr(node, "data-compteur"); return v == etat })
		if el == nil {
			t.Errorf("aucun compteur %q dans le bandeau", etat)
			continue
		}
		if got := text(el); got != n {
			t.Errorf("compteur %q = %q, attendu %q", etat, got, n)
		}
	}
}

// findElement renvoie le premier élément de la page qui vérifie match.
func findElement(t *testing.T, body string, match func(*html.Node) bool) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var find func(*html.Node) *html.Node
	find = func(n *html.Node) *html.Node {
		if n.Type == html.ElementNode && match(n) {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if found := find(c); found != nil {
				return found
			}
		}
		return nil
	}
	return find(doc)
}
