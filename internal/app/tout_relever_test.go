package app_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

// toutRelever appuie sur le bouton « Tout relever » du bandeau, comme le
// ferait le navigateur, et renvoie le code et la page obtenus.
func (s *service) toutRelever() (int, string) {
	s.t.Helper()
	return s.postForm("/releve", nil)
}

func TestToutReleverRelevesEachStoreOnce(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.ls("immich", response{Stdout: fixture(t, "ls-arr.txt")})
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, nil)
	svc.dashboard()
	clk.advance(5 * time.Minute)

	code, body := svc.toutRelever()

	if code != http.StatusOK {
		t.Fatalf("code %d après Tout relever\n%s", code, body)
	}
	for _, store := range []string{"arr", "immich"} {
		svc.waitRow(store, func(r row) bool { return r["releve"] == "2026-11-06 18:05" })
		if got := lsCalls(plakar, store); got != 2 {
			t.Errorf("%d Relevés de %s, attendu 2", got, store)
		}
	}
}

func TestToutReleverSkipsAStoreWhoseReleveIsInProgress(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"lent":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	// Le premier Relevé de lent n'est pas terminé pendant le test.
	plakar.ls("lent", response{DelayMS: 60_000})
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, nil)
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:00" })
	svc.waitRow("lent", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })
	clk.advance(5 * time.Minute)

	code, body := svc.toutRelever()

	if code != http.StatusOK {
		t.Fatalf("code %d après Tout relever\n%s", code, body)
	}
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:05" })
	if got := lsCalls(plakar, "lent"); got != 1 {
		t.Errorf("%d Relevés de lent, attendu 1 : Tout relever en a lancé un second", got)
	}
}

func TestToutReleverButtonIsDisabledWhenEveryStoreIsBeingReleved(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"lent":{}}` + "\n" + `{"tortue":{}}` + "\n")
	plakar.ls("lent", response{DelayMS: 60_000})
	plakar.ls("tortue", response{DelayMS: 60_000})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.waitRow("lent", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })
	svc.waitRow("tortue", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })

	_, body := svc.get("/")

	assertToutRelever(t, body, false)
}

func TestToutReleverButtonStaysEnabledWhileOneStoreIsNotBeingReleved(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"lent":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.ls("lent", response{DelayMS: 60_000})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:00" })
	svc.waitRow("lent", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })

	_, body := svc.get("/")

	assertToutRelever(t, body, true)
}

func TestPageRefreshesItselfOnlyWhileAReleveIsInProgress(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt"), DelayMS: 500})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.waitRow("arr", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })

	_, body := svc.get("/")
	_, rows, _ := parseDashboard(t, body)
	if rows["arr"]["releve-en-cours"] != "Relevé en cours" {
		t.Fatalf("le Relevé de arr est déjà terminé :\n%s", body)
	}
	if got, ok := refresh(t, body); !ok || got != "5" {
		t.Errorf("rafraîchissement = %q (présent : %v) pendant un Relevé, attendu « 5 »", got, ok)
	}

	svc.waitRow("arr", func(r row) bool { return r["releve-en-cours"] == "" })
	_, body = svc.get("/")

	if got, ok := refresh(t, body); ok {
		t.Errorf("rafraîchissement %q sans Relevé en cours, attendu aucun", got)
	}
}

func TestPageShowingAnActionErrorDoesNotRefreshItself(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"lent":{}}` + "\n")
	plakar.ls("lent", response{DelayMS: 60_000})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.waitRow("lent", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })

	// La page du refus est la réponse au POST : la recharger perdrait le
	// message.
	code, body := svc.setSeuil("lent", "bientôt")

	if code != http.StatusBadRequest {
		t.Fatalf("code %d, attendu %d\n%s", code, http.StatusBadRequest, body)
	}
	if got, ok := refresh(t, body); ok {
		t.Errorf("rafraîchissement %q sur la page d'un refus, attendu aucun", got)
	}
}

// refresh renvoie la période du rafraîchissement automatique de la page,
// <meta http-equiv="refresh" content="…">, et s'il y en a un.
func refresh(t *testing.T, body string) (string, bool) {
	t.Helper()
	meta := findElement(t, body, func(n *html.Node) bool {
		v, _ := attr(n, "http-equiv")
		return n.Data == "meta" && strings.EqualFold(v, "refresh")
	})
	if meta == nil {
		return "", false
	}
	content, _ := attr(meta, "content")
	return content, true
}

// assertToutRelever vérifie que le bandeau porte le bouton « Tout relever »,
// actif ou non.
func assertToutRelever(t *testing.T, body string, enabled bool) {
	t.Helper()
	bandeau := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-bandeau"); return ok })
	if bandeau == nil {
		t.Fatalf("aucun bandeau :\n%s", body)
	}
	button := findIn(bandeau, func(n *html.Node) bool { _, ok := attr(n, "data-tout-relever"); return ok })
	if button == nil {
		t.Fatalf("aucun bouton « Tout relever » dans le bandeau :\n%s", body)
	}
	if got := text(button); !strings.Contains(got, "Tout relever") {
		t.Errorf("bouton = %q, doit dire « Tout relever »", got)
	}
	if _, disabled := attr(button, "disabled"); disabled == enabled {
		t.Errorf("bouton « Tout relever » désactivé = %v, attendu %v", disabled, !enabled)
	}
}
