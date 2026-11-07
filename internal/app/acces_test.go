package app_test

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// oneStore démarre le service sur un seul Store, arr, avec ces variables
// d'environnement.
func oneStore(t *testing.T, env map[string]string) (*service, *fakePlakar) {
	t.Helper()
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	return startService(t, plakar, newClock("2026-11-06T17:00:00Z"), env), plakar
}

func TestRequestWithoutTailscaleIdentityIsForbidden(t *testing.T) {
	svc, _ := oneStore(t, nil)
	req := svc.request(http.MethodGet, "/")

	code, _ := svc.do(req, "")

	if code != http.StatusForbidden {
		t.Errorf("code %d, attendu %d", code, http.StatusForbidden)
	}
}

func TestUnknownTailscaleIdentityIsForbiddenOnEveryRoute(t *testing.T) {
	svc, plakar := oneStore(t, nil)
	svc.dashboard()

	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/"},
		{http.MethodPost, "/stores/arr/releve"},
		{http.MethodPost, "/stores/arr/seuil"},
		// Une route ajoutée plus tard est couverte d'office.
		{http.MethodGet, "/route-a-venir"},
	} {
		req := svc.request(route.method, route.path)
		if code, _ := svc.do(req, "intrus@example.com"); code != http.StatusForbidden {
			t.Errorf("%s %s : code %d, attendu %d", route.method, route.path, code, http.StatusForbidden)
		}
	}
	_, rows := svc.dashboard()
	if got := lsCalls(plakar, "arr"); got != 1 {
		t.Errorf("%d Relevés de arr, attendu 1 : le ↻ refusé n'a rien lancé", got)
	}
	assertRow(t, rows["arr"], row{"seuil": "26h (par défaut)"})
}

func TestAuthorizedUsersAreReadFromEnvironment(t *testing.T) {
	svc, _ := oneStore(t, map[string]string{
		"PLAKAR_UI_UTILISATEURS": "alice@example.com, bob@example.com",
	})

	for _, user := range []string{"alice@example.com", "bob@example.com", "Bob@Example.com"} {
		req := svc.request(http.MethodGet, "/")
		if code, _ := svc.do(req, user); code != http.StatusOK {
			t.Errorf("%s : code %d, attendu %d", user, code, http.StatusOK)
		}
	}
	req := svc.request(http.MethodGet, "/")
	if code, _ := svc.do(req, testUser); code != http.StatusForbidden {
		t.Errorf("%s hors de la liste : code %d, attendu %d", testUser, code, http.StatusForbidden)
	}
}

func TestServiceRefusesToStartWithoutAuthorizedUsers(t *testing.T) {
	for _, users := range []string{"", " , "} {
		plakar := newFakePlakar(t)
		plakar.stores("")
		a, err := tryStartService(plakar, newClock("2026-11-06T17:00:00Z"),
			map[string]string{"PLAKAR_UI_UTILISATEURS": users})
		if err == nil {
			a.Close()
			t.Fatalf("PLAKAR_UI_UTILISATEURS=%q : le service a démarré sans Utilisateur autorisé", users)
		}
		if !strings.Contains(err.Error(), "PLAKAR_UI_UTILISATEURS") {
			t.Errorf("erreur = %q, doit nommer la variable à renseigner", err)
		}
	}
}

func TestOnlyExpectedHostsAreServedWhateverThePort(t *testing.T) {
	svc, _ := oneStore(t, map[string]string{"PLAKAR_UI_HOTE": "plakar-ui.tailnet.ts.net"})

	for host, want := range map[string]int{
		"localhost":                    http.StatusOK,
		"localhost:8080":               http.StatusOK,
		"127.0.0.1":                    http.StatusOK,
		"127.0.0.1:9300":               http.StatusOK,
		"[::1]":                        http.StatusOK,
		"[::1]:8080":                   http.StatusOK,
		"plakar-ui.tailnet.ts.net":     http.StatusOK,
		"plakar-ui.tailnet.ts.net:443": http.StatusOK,
		"PLAKAR-UI.tailnet.ts.net":     http.StatusOK,
		// Un nom qu'une page malveillante a fait pointer vers 127.0.0.1.
		"evil.example":                   http.StatusForbidden,
		"evil.example:8080":              http.StatusForbidden,
		"plakar-ui.tailnet.ts.net.evil":  http.StatusForbidden,
		"autre-noeud.tailnet.ts.net:443": http.StatusForbidden,
	} {
		req := svc.request(http.MethodGet, "/")
		req.Host = host
		if code, _ := svc.do(req, testUser); code != want {
			t.Errorf("Host %q : code %d, attendu %d", host, code, want)
		}
	}
}

func TestUnexpectedHostIsRefusedBeforeTheIdentityCheck(t *testing.T) {
	svc, _ := oneStore(t, nil)
	req := svc.request(http.MethodGet, "/")
	req.Host = "evil.example"

	// Une page malveillante peut poser elle-même l'en-tête d'identité.
	code, body := svc.do(req, "")

	if code != http.StatusForbidden {
		t.Errorf("code %d, attendu %d", code, http.StatusForbidden)
	}
	if !strings.Contains(body, "nom d'hôte") {
		t.Errorf("réponse = %q, doit refuser le nom d'hôte avant de regarder l'identité", body)
	}
}

func TestDashboardShowsTheIdentityOfTheAuthorizedUser(t *testing.T) {
	svc, _ := oneStore(t, map[string]string{
		"PLAKAR_UI_UTILISATEURS": "alice@example.com,bob@example.com",
	})
	req := svc.request(http.MethodGet, "/")

	_, body := svc.do(req, "bob@example.com")

	if got := elementText(t, body, "utilisateur"); got != "bob@example.com" {
		t.Errorf("identité affichée = %q, attendu %q", got, "bob@example.com")
	}
}

// elementText renvoie le texte de l'élément de la page qui porte cet id.
func elementText(t *testing.T, body, id string) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var find func(*html.Node) *html.Node
	find = func(n *html.Node) *html.Node {
		if v, ok := attr(n, "id"); ok && n.Type == html.ElementNode && v == id {
			return n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if found := find(c); found != nil {
				return found
			}
		}
		return nil
	}
	n := find(doc)
	if n == nil {
		t.Fatalf("aucun élément id=%q :\n%s", id, body)
	}
	return text(n)
}

func TestServiceRefusesAPublicHostThatNoRequestCouldMatch(t *testing.T) {
	for _, hote := range []string{"https://plakar-ui.tailnet.ts.net", "plakar-ui.tailnet.ts.net:443", "plakar-ui.tailnet.ts.net/"} {
		plakar := newFakePlakar(t)
		plakar.stores("")
		a, err := tryStartService(plakar, newClock("2026-11-06T17:00:00Z"),
			map[string]string{"PLAKAR_UI_HOTE": hote})
		if err == nil {
			a.Close()
			t.Errorf("PLAKAR_UI_HOTE=%q accepté : toute requête vers ce nom serait refusée", hote)
			continue
		}
		if !strings.Contains(err.Error(), "PLAKAR_UI_HOTE") {
			t.Errorf("erreur = %q, doit nommer la variable", err)
		}
	}
}

func TestPublicHostMayBeWrittenAsAnAbsoluteName(t *testing.T) {
	svc, _ := oneStore(t, map[string]string{"PLAKAR_UI_HOTE": "plakar-ui.tailnet.ts.net."})
	req := svc.request(http.MethodGet, "/")
	req.Host = "plakar-ui.tailnet.ts.net:443"

	if code, _ := svc.do(req, testUser); code != http.StatusOK {
		t.Errorf("code %d, attendu %d", code, http.StatusOK)
	}
}
