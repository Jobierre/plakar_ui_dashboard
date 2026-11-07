package app_test

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestStoreCommandOutputAndExitCodeAreShown(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.on([]string{"store", "show", "arr"}, response{
		Stdout: "arr:\n    location: http://unraid:9876\n    passphrase: '********'\n",
		Stderr: "avertissement\n",
		Code:   0,
	})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	code, body := svc.commande("plakar store show arr")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	got := commandeBlock(t, body)
	want := map[string]string{
		"stdout": "arr:\n    location: http://unraid:9876\n    passphrase: '********'\n",
		"stderr": "avertissement\n",
		"code":   "0",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, attendu %q", k, got[k], v)
		}
	}
}

// commandeCalls renvoie les appels reçus par le faux plakar, hors démarrage et
// Relevés.
func commandeCalls(plakar *fakePlakar) [][]string {
	var got [][]string
	for _, c := range plakar.calls() {
		if len(c) > 0 && c[0] == "at" {
			continue
		}
		if len(c) == 3 && c[0] == "store" && c[1] == "show" && c[2] == "-json" {
			continue
		}
		got = append(got, c)
	}
	return got
}

func TestStoreCommandsOutsideTheAllowListAreRefusedWithoutRunningPlakar(t *testing.T) {
	refused := map[string]string{
		"vide":                   "",
		"blancs":                 "   ",
		"plakar seul":            "plakar",
		"store seul":             "plakar store",
		"autre commande système": "rm -rf /",
		"autre commande plakar":  "plakar backup /etc",
		"chemin vers plakar":     "/usr/bin/plakar store show",
		"option globale":         "plakar -config /tmp store show",
		"store ls":               "plakar store ls",
		"store import":           "store import -config /tmp/x.yml",
		"store check":            "plakar store check arr",
		"store ping":             "plakar store ping arr",
		"sous-commande inconnue": "plakar store dump",
		"-secrets":               "plakar store show -secrets",
		"--secrets":              "plakar store show --secrets",
		"-secrets=true":          "plakar store show -secrets=true",
		"--secrets=1":            "store show --secrets=1 arr",
		"-secrets après le nom":  "plakar store show arr -secrets",
		"-secrets sur add":       "plakar store add x -secrets",
		"majuscules":             "plakar store show -SECRETS",
		"entre guillemets":       `plakar store show "-secrets"`,
		"guillemet non fermé":    `plakar store add x passphrase="abc`,
		"point-virgule collé":    "plakar store show; id",
	}
	for name, line := range refused {
		t.Run(name, func(t *testing.T) {
			plakar := newFakePlakar(t)
			plakar.stores(`{"arr":{}}` + "\n")
			plakar.ls("arr", response{})
			svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
			svc.dashboard()

			code, body := svc.commande(line)

			if code != http.StatusBadRequest {
				t.Errorf("code %d, attendu %d", code, http.StatusBadRequest)
			}
			got := commandeBlock(t, body)
			if got["refus"] == "" {
				t.Errorf("aucun message de refus :\n%s", body)
			}
			if _, ok := got["code"]; ok {
				t.Errorf("une sortie est affichée pour une commande refusée")
			}
			if calls := commandeCalls(plakar); len(calls) != 0 {
				t.Errorf("plakar a été appelé : %q", calls)
			}
		})
	}
}

func TestSecretsRefusalSaysWhy(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores("")
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	_, body := svc.commande("plakar store show -secrets")

	if got := commandeBlock(t, body)["refus"]; !strings.Contains(got, "-secrets") {
		t.Errorf("refus = %q, doit nommer -secrets", got)
	}
}

func TestAllowedStoreCommandsReachPlakarAsGiven(t *testing.T) {
	accepted := map[string][]string{
		"show":                     {"plakar store show", "store", "show"},
		"show sans préfixe":        {"store show -json arr", "store", "show", "-json", "arr"},
		"add":                      {"plakar store add neuf location=http://unraid:9880", "store", "add", "neuf", "location=http://unraid:9880"},
		"set":                      {"plakar store set arr location=http://unraid:9881", "store", "set", "arr", "location=http://unraid:9881"},
		"unset":                    {"plakar store unset arr location", "store", "unset", "arr", "location"},
		"rm":                       {"plakar store rm arr", "store", "rm", "arr"},
		"blancs multiples":         {"  plakar   store\tshow  ", "store", "show"},
		"valeur contenant secrets": {"plakar store set arr passphrase=secrets", "store", "set", "arr", "passphrase=secrets"},
	}
	for name, c := range accepted {
		t.Run(name, func(t *testing.T) {
			plakar := newFakePlakar(t)
			plakar.stores(`{"arr":{}}` + "\n")
			plakar.ls("arr", response{})
			plakar.on(c[1:], response{Code: 1, Stderr: "refusé par le faux plakar\n"})
			svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
			svc.dashboard()

			code, body := svc.commande(c[0])

			if code != http.StatusOK {
				t.Fatalf("code %d\n%s", code, body)
			}
			if got := commandeBlock(t, body)["code"]; got != "1" {
				t.Errorf("code de retour affiché %q, attendu 1", got)
			}
			if calls := commandeCalls(plakar); len(calls) != 1 || !slices.Equal(calls[0], c[1:]) {
				t.Errorf("appels %q, attendu %q", calls, c[1:])
			}
		})
	}
}

func TestQuotedValuesAndShellCharactersReachPlakarLiterally(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "pwn")
	cases := map[string][]string{
		"passphrase entre guillemets doubles": {
			`plakar store add neuf location=http://unraid:9880 passphrase="ma phrase  secrète"`,
			"store", "add", "neuf", "location=http://unraid:9880", "passphrase=ma phrase  secrète",
		},
		"passphrase entre guillemets simples": {
			`store set arr passphrase='a "b" $HOME \n'`,
			"store", "set", "arr", `passphrase=a "b" $HOME \n`,
		},
		"échappements": {
			`store set arr passphrase="un \"deux\" \\ trois" quatre\ cinq`,
			"store", "set", "arr", `passphrase=un "deux" \ trois`, "quatre cinq",
		},
		"argument vide": {
			`store set arr passphrase=""`,
			"store", "set", "arr", "passphrase=",
		},
		"caractères de shell": {
			"plakar store set arr location=x;touch " + marker + " | touch " + marker + " && $(touch " + marker + ") `touch " + marker + "`",
			"store", "set", "arr", "location=x;touch", marker, "|", "touch", marker, "&&", "$(touch", marker + ")", "`touch", marker + "`",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			plakar := newFakePlakar(t)
			plakar.stores(`{"arr":{}}` + "\n")
			plakar.ls("arr", response{})
			plakar.on(c[1:], response{Code: 1})
			svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
			svc.dashboard()

			code, body := svc.commande(c[0])

			if code != http.StatusOK {
				t.Fatalf("code %d\n%s", code, body)
			}
			if calls := commandeCalls(plakar); len(calls) != 1 || !slices.Equal(calls[0], c[1:]) {
				t.Errorf("appels %q, attendu %q", calls, c[1:])
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Errorf("une commande de shell a été exécutée (%v)", err)
			}
		})
	}
}

// storeShowArgs sont les arguments de la lecture du Catalogue.
var storeShowArgs = []string{"store", "show", "-json"}

func TestAddedStoreAppearsAtOnceAndIsRelevedImmediately(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.on([]string{"store", "add", "neuf", "location=http://unraid:9880"}, response{
		Replace: []response{{Args: storeShowArgs, Stdout: `{"neuf":{}}` + "\n" + `{"arr":{}}` + "\n"}},
	})
	plakar.ls("neuf", response{Stdout: fixture(t, "ls-small.txt"), DelayMS: 300})
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, nil)
	svc.dashboard()

	code, body := svc.commande("plakar store add neuf location=http://unraid:9880")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	names, rows, _ := parseDashboard(t, body)
	if !slices.Equal(names, []string{"arr", "neuf"}) {
		t.Fatalf("Stores affichés %q, attendu arr puis neuf", names)
	}
	assertRow(t, rows["neuf"], row{"releve-en-cours": "Relevé en cours"})
	got := svc.waitRow("neuf", func(r row) bool { return r["releve"] != "" })
	if got["id"] == "" {
		t.Errorf("le Relevé de neuf n'a pas abouti : %v", got)
	}
	if n := lsCalls(plakar, "neuf"); n != 1 {
		t.Errorf("%d Relevés de neuf, attendu 1", n)
	}
	// Le Store ajouté est ensuite relevé chaque heure, comme les autres.
	clk.advance(time.Hour)
	svc.waitRow("neuf", func(r row) bool { return r["releve"] == "2026-11-06 19:00" })
}

func TestRemovedStoreDisappearsWithItsStateAndIsNoLongerReleved(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.ls("immich", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.on([]string{"store", "rm", "immich"}, response{
		Replace: []response{{Args: storeShowArgs, Stdout: `{"arr":{}}` + "\n"}},
	})
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, map[string]string{"PLAKAR_UI_STATE": statePath})
	svc.dashboard()
	svc.setSeuil("immich", "8d")

	code, body := svc.commande("store rm immich")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	if names, _, _ := parseDashboard(t, body); !slices.Equal(names, []string{"arr"}) {
		t.Errorf("Stores affichés %q, attendu arr seul", names)
	}
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "immich") {
		t.Errorf("le fichier d'état garde immich :\n%s", raw)
	}
	clk.advance(time.Hour)
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 19:00" })
	if n := lsCalls(plakar, "immich"); n != 1 {
		t.Errorf("%d Relevés de immich, attendu 1 : le Store retiré est encore relevé", n)
	}
	if strings.Contains(svc.log.String(), "level=ERROR") {
		t.Errorf("erreur journalisée après le retrait :\n%s", svc.log)
	}
}

func TestFailedStoreCommandLeavesTheCatalogueAsIs(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	plakar.on([]string{"store", "rm", "inconnu"}, response{Stderr: "plakar: store inconnu does not exist\n", Code: 1})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()

	_, body := svc.commande("store rm inconnu")

	got := commandeBlock(t, body)
	if got["code"] != "1" || got["stderr"] != "plakar: store inconnu does not exist\n" {
		t.Errorf("bloc %v", got)
	}
	n := 0
	for _, c := range plakar.calls() {
		if slices.Equal(c, storeShowArgs) {
			n++
		}
	}
	if n != 1 {
		t.Errorf("Catalogue relu %d fois, attendu 1 (au démarrage seulement)", n)
	}
}

func TestPassphraseOfAStoreCommandIsNeitherLoggedNorPersisted(t *testing.T) {
	const passphrase = "correct-horse-battery-staple"
	statePath := filepath.Join(t.TempDir(), "state.json")
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	// plakar échoue et cite la passphrase, puis réussit, pour couvrir les
	// deux chemins.
	plakar.on([]string{"store", "set", "arr", "passphrase=" + passphrase + " x"}, response{
		Stderr: "plakar: invalid value passphrase=" + passphrase + " x\n", Code: 1,
	})
	plakar.on([]string{"store", "add", "neuf", "passphrase=" + passphrase}, response{
		Stdout:  "ajouté avec " + passphrase + "\n",
		Replace: []response{{Args: storeShowArgs, Stdout: `{"arr":{}}` + "\n" + `{"neuf":{}}` + "\n"}},
	})
	plakar.ls("neuf", response{Stderr: "plakar: bad passphrase\n", Code: 1})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), map[string]string{"PLAKAR_UI_STATE": statePath})
	svc.dashboard()

	svc.commande(`plakar store set arr passphrase="` + passphrase + ` x"`)
	svc.commande("plakar store add neuf passphrase=" + passphrase)
	svc.commande(`plakar store show -secrets passphrase=` + passphrase)
	svc.waitRow("neuf", func(r row) bool { return r["error"] != "" })
	svc.stop()

	if strings.Contains(svc.log.String(), passphrase) {
		t.Errorf("la passphrase est dans les journaux :\n%s", svc.log)
	}
	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), passphrase) {
		t.Errorf("la passphrase est dans le fichier d'état :\n%s", raw)
	}
	if !strings.Contains(svc.log.String(), "neuf") {
		t.Errorf("journaux vides : le test ne prouve rien\n%s", svc.log)
	}
}

func TestStoreCommandPageIsNotCachedAndDoesNotEchoTheCommand(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores("")
	plakar.on([]string{"store", "set", "arr", "passphrase=s3cr3t-écho"}, response{Code: 1})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	req := svc.request(http.MethodPost, "/commande")
	req.Body = io.NopCloser(strings.NewReader(url.Values{"commande": {"store set arr passphrase=s3cr3t-écho"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(withIdentity(req))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, attendu no-store", got)
	}
	if strings.Contains(string(body), "s3cr3t") {
		t.Errorf("la page renvoie la commande saisie :\n%s", body)
	}
}

func TestStoreCommandIsRefusedToOthersThanAuthorizedUsers(t *testing.T) {
	for name, user := range map[string]string{"sans identité": "", "identité inconnue": "intrus@example.com"} {
		t.Run(name, func(t *testing.T) {
			plakar := newFakePlakar(t)
			plakar.stores("")
			plakar.on([]string{"store", "rm", "arr"}, response{})
			svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
			req := svc.request(http.MethodPost, "/commande")
			req.Body = io.NopCloser(strings.NewReader("commande=store+rm+arr"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			code, _ := svc.do(req, user)

			if code != http.StatusForbidden {
				t.Errorf("code %d, attendu %d", code, http.StatusForbidden)
			}
			if calls := commandeCalls(plakar); len(calls) != 0 {
				t.Errorf("plakar a été appelé : %q", calls)
			}
		})
	}
}

func TestStoreCommandIsRefusedFromAnotherSite(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores("")
	plakar.on([]string{"store", "rm", "arr"}, response{})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	req := svc.request(http.MethodPost, "/commande")
	req.Body = io.NopCloser(strings.NewReader("commande=store+rm+arr"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	code, _ := svc.do(req, testUser)

	if code != http.StatusForbidden {
		t.Errorf("code %d, attendu %d", code, http.StatusForbidden)
	}
	if calls := commandeCalls(plakar); len(calls) != 0 {
		t.Errorf("plakar a été appelé : %q", calls)
	}
}

// withIdentity pose l'identité de l'Utilisateur autorisé des tests.
func withIdentity(req *http.Request) *http.Request {
	req.Header.Set(identityHeader, testUser)
	return req
}

func TestStoreRemovedAndAddedAgainDuringItsReleveIsRelevedWithItsNewConfig(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	// L'ancien Kloset de arr est lent : son Relevé est encore en cours.
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt"), DelayMS: 1500})
	plakar.on([]string{"store", "rm", "arr"}, response{
		Replace: []response{{Args: storeShowArgs, Stdout: ""}},
	})
	plakar.on([]string{"store", "add", "arr", "location=/data/petit"}, response{
		Replace: []response{
			{Args: storeShowArgs, Stdout: `{"arr":{}}` + "\n"},
			{Args: []string{"at", "@arr", "ls", "-tags"}, Stdout: fixture(t, "ls-small.txt")},
		},
	})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.waitRow("arr", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })

	svc.commande("store rm arr")
	_, body := svc.commande("store add arr location=/data/petit")

	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"releve-en-cours": "Relevé en cours"})
	svc.waitRow("arr", func(r row) bool { return r["id"] == "19ae0c83" })
	// Le Relevé de l'ancienne config, s'il allait à son terme, ne remplace
	// pas celui de la nouvelle.
	time.Sleep(1700 * time.Millisecond)
	_, rows = svc.dashboard()
	assertRow(t, rows["arr"], row{"id": "19ae0c83"})
	if n := lsCalls(plakar, "arr"); n != 2 {
		t.Errorf("%d Relevés de arr, attendu 2", n)
	}
}

func TestStoreCommandGoesToItsEndWhenTheBrowserGoesAway(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores("")
	plakar.on([]string{"store", "add", "neuf", "location=/data/petit"}, response{
		DelayMS: 500,
		Replace: []response{{Args: storeShowArgs, Stdout: `{"neuf":{}}` + "\n"}},
	})
	plakar.ls("neuf", response{Stdout: fixture(t, "ls-small.txt")})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	req := svc.request(http.MethodPost, "/commande")
	req.Body = io.NopCloser(strings.NewReader(url.Values{"commande": {"store add neuf location=/data/petit"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Le navigateur abandonne avant la fin de plakar.
	impatient := &http.Client{Timeout: 100 * time.Millisecond}
	if resp, err := impatient.Do(withIdentity(req)); err == nil {
		resp.Body.Close()
		t.Fatal("la requête devait être abandonnée")
	}

	svc.waitRow("neuf", func(r row) bool { return r["id"] == "19ae0c83" })
}
