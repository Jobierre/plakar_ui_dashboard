package app_test

import (
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// explorationEnv est l'environnement d'un test d'Exploration : une plage de
// trois ports publics libres, à partir de base.
func explorationEnv(base int, extra map[string]string) map[string]string {
	env := map[string]string{"PLAKAR_UI_PORTS": fmt.Sprintf("%d-%d", base, base+2)}
	for k, v := range extra {
		env[k] = v
	}
	return env
}

func TestExploreLaunchesPlakarUIAndRelaysItsPublicPort(t *testing.T) {
	base := freePorts(3)
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	plakar.ui("arr", base, "tok-1")
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"),
		explorationEnv(base, map[string]string{"PLAKAR_UI_HOTE": "plakar-ui.example.ts.net"}))
	svc.dashboard()

	code, body := svc.explorer("arr")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	_, rows, _ := parseDashboard(t, body)
	want := fmt.Sprintf("https://plakar-ui.example.ts.net:%d/?plakar_token=tok-1", base)
	assertRow(t, rows["arr"], row{"exploration": want})
	if calls := plakar.calls(); !slices.ContainsFunc(calls, func(c []string) bool { return slices.Equal(c, uiArgs("arr", base)) }) {
		t.Errorf("appels %q, attendu %q", calls, uiArgs("arr", base))
	}
	code, page := svc.getPort(base, "/snapshots?plakar_token=tok-1", testUser)
	if code != http.StatusOK || page != "arr GET /snapshots?plakar_token=tok-1" {
		t.Errorf("port public : code %d, page %q", code, page)
	}
}

// uiPID renvoie le PID du dernier `plakar ui` lancé par le faux plakar.
func uiPID(t *testing.T, plakar *fakePlakar) int {
	t.Helper()
	calls, pids := plakar.calls(), plakar.pids()
	for i := len(calls) - 1; i >= 0; i-- {
		if len(calls[i]) > 2 && calls[i][2] == "ui" {
			return pids[i]
		}
	}
	t.Fatal("aucun plakar ui lancé")
	return 0
}

// assertGone vérifie que le processus est terminé.
func assertGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			t.Fatalf("le processus plakar ui %d est encore là", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func uiCalls(plakar *fakePlakar) int {
	n := 0
	for _, c := range plakar.calls() {
		if len(c) > 2 && c[2] == "ui" {
			n++
		}
	}
	return n
}

func TestEachStoreGetsTheFirstFreePortWhichSurvivesARestartAndIsFreedOnRemoval(t *testing.T) {
	base := freePorts(3)
	state := filepath.Join(t.TempDir(), "state.json")
	env := explorationEnv(base, map[string]string{"PLAKAR_UI_STATE": state})
	clk := newClock("2026-11-06T17:00:05Z")
	before := newFakePlakar(t)
	before.stores(`{"immich":{}}` + "\n")
	before.ls("immich", response{})
	svc := startService(t, before, clk, env)
	svc.dashboard()
	svc.stop()

	// arr arrive après immich : immich garde son port, arr prend le suivant.
	after := newFakePlakar(t)
	after.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	after.ls("arr", response{})
	after.ls("immich", response{})
	after.ui("arr", base+1, "tok-arr")
	after.ui("immich", base, "tok-immich")
	after.on([]string{"store", "rm", "immich"}, response{Replace: []response{{Args: storeShowArgs, Stdout: `{"arr":{}}` + "\n"}}})
	after.on([]string{"store", "add", "neuf", "location=/x"}, response{Replace: []response{{Args: storeShowArgs, Stdout: `{"arr":{}}` + "\n" + `{"neuf":{}}` + "\n"}}})
	after.ls("neuf", response{})
	after.ui("neuf", base, "tok-neuf")
	svc = startService(t, after, clk, env)
	svc.dashboard()

	svc.explorer("arr")
	_, body := svc.explorer("immich")
	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"exploration": fmt.Sprintf("http://localhost:%d/?plakar_token=tok-arr", base+1)})
	assertRow(t, rows["immich"], row{"exploration": fmt.Sprintf("http://localhost:%d/?plakar_token=tok-immich", base)})

	// Le port de immich, retiré, revient au Store ajouté.
	svc.commande("store rm immich")
	svc.commande("store add neuf location=/x")
	_, body = svc.explorer("neuf")
	_, rows, _ = parseDashboard(t, body)
	assertRow(t, rows["neuf"], row{"exploration": fmt.Sprintf("http://localhost:%d/?plakar_token=tok-neuf", base)})
}

func TestStoreWithoutPortWhenTheRangeIsExhaustedShowsAClearMessage(t *testing.T) {
	base := freePorts(3)
	plakar := newFakePlakar(t)
	plakar.stores(`{"a":{}}` + "\n" + `{"b":{}}` + "\n")
	plakar.ls("a", response{})
	plakar.ls("b", response{})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"),
		map[string]string{"PLAKAR_UI_PORTS": fmt.Sprintf("%d-%d", base, base)})

	_, rows := svc.dashboard()

	assertRow(t, rows["a"], row{"port-message": ""})
	if !strings.Contains(rows["b"]["port-message"], "PLAKAR_UI_PORTS") {
		t.Errorf("message %q, doit nommer PLAKAR_UI_PORTS", rows["b"]["port-message"])
	}
	if code, _ := svc.explorer("b"); code != http.StatusConflict {
		t.Errorf("Explorer sans port : code %d, attendu %d", code, http.StatusConflict)
	}
}

func TestExploreTwiceGivesTheSameLinkWithoutASecondProcess(t *testing.T) {
	base := freePorts(3)
	svc, plakar := exploringArr(t, base, nil)

	_, body := svc.explorer("arr")

	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"exploration": fmt.Sprintf("http://localhost:%d/?plakar_token=tok-1", base)})
	if n := uiCalls(plakar); n != 1 {
		t.Errorf("%d plakar ui lancés, attendu 1", n)
	}
}

// exploringArr démarre le service sur arr, et lance son Exploration.
func exploringArr(t *testing.T, base int, extra map[string]string) (*service, *fakePlakar) {
	t.Helper()
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	plakar.ui("arr", base, "tok-1")
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), explorationEnv(base, extra))
	svc.dashboard()
	if code, body := svc.explorer("arr"); code != http.StatusOK {
		t.Fatalf("Explorer : code %d\n%s", code, body)
	}
	return svc, plakar
}

func TestStopButtonEndsTheExploration(t *testing.T) {
	base := freePorts(3)
	svc, plakar := exploringArr(t, base, nil)
	pid := uiPID(t, plakar)

	code, body := svc.arreterExploration("arr")

	if code != http.StatusOK {
		t.Fatalf("code %d\n%s", code, body)
	}
	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"exploration": ""})
	assertGone(t, pid)
	if code, _ := svc.getPort(base, "/?plakar_token=tok-1", testUser); code != http.StatusNotFound {
		t.Errorf("port public après arrêt : code %d, attendu %d", code, http.StatusNotFound)
	}
}

func TestExplorationPortRefusesOthersThanAuthorizedUsers(t *testing.T) {
	base := freePorts(3)
	svc, _ := exploringArr(t, base, nil)

	for _, user := range []string{"", "intrus@example.com"} {
		code, page := svc.getPort(base, "/?plakar_token=tok-1", user)
		if code != http.StatusForbidden || strings.Contains(page, "arr GET") {
			t.Errorf("identité %q : code %d, page %q", user, code, page)
		}
	}
}

func TestFailedLaunchShowsPlakarMessage(t *testing.T) {
	base := freePorts(3)
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	plakar.on(uiArgs("arr", base), response{Stderr: "plakar: failed to open repository: bad passphrase\n", Code: 1})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), explorationEnv(base, nil))
	svc.dashboard()

	code, body := svc.explorer("arr")

	if code != http.StatusBadGateway {
		t.Errorf("code %d, attendu %d", code, http.StatusBadGateway)
	}
	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"exploration-error": "plakar: failed to open repository: bad passphrase", "exploration": ""})
}

func TestRemovingAStoreStopsItsExploration(t *testing.T) {
	base := freePorts(3)
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	plakar.ui("arr", base, "tok-1")
	plakar.on([]string{"store", "rm", "arr"}, response{Replace: []response{{Args: storeShowArgs, Stdout: ""}}})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), explorationEnv(base, nil))
	svc.dashboard()
	svc.explorer("arr")
	pid := uiPID(t, plakar)

	svc.commande("store rm arr")

	assertGone(t, pid)
	if code, _ := svc.getPort(base, "/?plakar_token=tok-1", testUser); code != http.StatusNotFound {
		t.Errorf("port public après retrait : code %d, attendu %d", code, http.StatusNotFound)
	}
}

func TestExplorationStopsAfterInactivityButNotWhileInUse(t *testing.T) {
	base := freePorts(3)
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	plakar.ui("arr", base, "tok-1")
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, explorationEnv(base, map[string]string{"PLAKAR_UI_INACTIVITE": "10m"}))
	svc.dashboard()
	svc.explorer("arr")
	pid := uiPID(t, plakar)

	// Du trafic à 8 min repousse l'arrêt.
	clk.advance(8 * time.Minute)
	if code, _ := svc.getPort(base, "/?plakar_token=tok-1", testUser); code != http.StatusOK {
		t.Fatalf("port public : code %d", code)
	}
	clk.advance(8 * time.Minute)
	time.Sleep(100 * time.Millisecond)
	if syscall.Kill(pid, 0) != nil {
		t.Fatal("Exploration arrêtée malgré le trafic")
	}
	_, rows := svc.dashboard()
	assertRow(t, rows["arr"], row{"exploration": fmt.Sprintf("http://localhost:%d/?plakar_token=tok-1", base)})

	clk.advance(2 * time.Minute)

	assertGone(t, pid)
	_, rows = svc.dashboard()
	assertRow(t, rows["arr"], row{"exploration": ""})
}

func TestExplorationTokenIsNeverLogged(t *testing.T) {
	base := freePorts(3)
	svc, _ := exploringArr(t, base, nil)
	svc.getPort(base, "/?plakar_token=tok-1", testUser)
	svc.arreterExploration("arr")
	svc.stop()

	if !strings.Contains(svc.log.String(), "Exploration lancée") {
		t.Fatalf("journaux sans lancement : le test ne prouve rien\n%s", svc.log)
	}
	if strings.Contains(svc.log.String(), "tok-1") {
		t.Errorf("le jeton est dans les journaux :\n%s", svc.log)
	}
}

func TestServiceRefusesInvalidExplorationSettings(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"plage illisible":         {"PLAKAR_UI_PORTS": "9300"},
		"plage à l'envers":        {"PLAKAR_UI_PORTS": "9320-9300"},
		"décalage dans la plage":  {"PLAKAR_UI_PORTS": "20000-20020", "PLAKAR_UI_DECALAGE": "10"},
		"décalage hors des ports": {"PLAKAR_UI_PORTS": "65000-65010", "PLAKAR_UI_DECALAGE": "1000"},
		"inactivité nulle":        {"PLAKAR_UI_INACTIVITE": "0s"},
		"inactivité illisible":    {"PLAKAR_UI_INACTIVITE": "souvent"},
	} {
		t.Run(name, func(t *testing.T) {
			plakar := newFakePlakar(t)
			plakar.stores("")
			a, err := tryStartService(plakar, newClock("2026-11-06T17:00:05Z"), env)
			if err == nil {
				a.Close()
				t.Fatal("le service a démarré")
			}
			for k := range env {
				if strings.Contains(err.Error(), k) {
					return
				}
			}
			t.Errorf("erreur %q : doit nommer la variable", err)
		})
	}
}

func TestRemovingAStoreDuringItsLaunchStopsIt(t *testing.T) {
	base := freePorts(3)
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	// plakar ui met du temps à ouvrir le Store.
	plakar.on(uiArgs("arr", base), response{
		Serve: true, Token: "tok-1", DelayMS: 2000,
		Stdout: fmt.Sprintf("launching webUI at http://127.0.0.1:%d?plakar_token=tok-1\n", base+100),
	})
	plakar.on([]string{"store", "rm", "arr"}, response{Replace: []response{{Args: storeShowArgs, Stdout: ""}}})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), explorationEnv(base, nil))
	svc.dashboard()
	launched := make(chan int, 1)
	go func() {
		code, _ := svc.explorer("arr")
		launched <- code
	}()
	deadline := time.Now().Add(5 * time.Second)
	for uiCalls(plakar) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("plakar ui jamais lancé")
		}
		time.Sleep(10 * time.Millisecond)
	}
	pid := uiPID(t, plakar)

	start := time.Now()
	svc.commande("store rm arr")

	assertGone(t, pid)
	if d := time.Since(start); d > 1500*time.Millisecond {
		t.Errorf("le retrait a attendu la fin du lancement (%s)", d)
	}
	if code := <-launched; code != http.StatusBadGateway && code != http.StatusNotFound {
		t.Errorf("Explorer interrompu : code %d", code)
	}
}

func TestStoppingTheServiceStopsItsExplorations(t *testing.T) {
	base := freePorts(3)
	svc, plakar := exploringArr(t, base, nil)
	pid := uiPID(t, plakar)

	svc.stop()

	assertGone(t, pid)
}

// Le vrai `plakar ui` imprime sa ligne de lancement avant d'ouvrir son port :
// le lien n'est donné qu'une fois le port ouvert, sans quoi le premier clic
// tombe sur une erreur du relais.
func TestExploreWaitsForPlakarUIToListenBeforeGivingTheLink(t *testing.T) {
	base := freePorts(3)
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	plakar.uiListeningAfter("arr", base, "tok-1", 500*time.Millisecond)
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), explorationEnv(base, nil))
	svc.dashboard()

	if code, body := svc.explorer("arr"); code != http.StatusOK {
		t.Fatalf("Explorer : code %d\n%s", code, body)
	}

	code, page := svc.getPort(base, "/?plakar_token=tok-1", testUser)
	if code != http.StatusOK || page != "arr GET /?plakar_token=tok-1" {
		t.Errorf("premier accès au lien : code %d, page %q", code, page)
	}
}
