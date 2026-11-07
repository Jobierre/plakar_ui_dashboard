package app_test

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// lsCalls compte les Relevés du Store reçus par le faux plakar.
func lsCalls(plakar *fakePlakar, store string) int {
	n := 0
	for _, c := range plakar.calls() {
		if len(c) == 4 && c[0] == "at" && c[1] == "@"+store && c[2] == "ls" {
			n++
		}
	}
	return n
}

func TestEachStoreIsRelevedAgainEveryHourByDefault(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, nil)
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:00" })

	clk.advance(59 * time.Minute)
	svc.dashboard()
	if got := lsCalls(plakar, "arr"); got != 1 {
		t.Fatalf("%d Relevés avant l'heure, attendu 1", got)
	}

	clk.advance(time.Minute)

	got := svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 19:00" })
	assertRow(t, got, row{"releve-en-cours": "", "id": "93ebf1a2"})
	if got := lsCalls(plakar, "arr"); got != 2 {
		t.Errorf("%d Relevés après une heure, attendu 2", got)
	}
}

func TestReleveIntervalIsSetByEnvironment(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, map[string]string{"PLAKAR_UI_INTERVALLE": "15m"})
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:00" })

	clk.advance(15 * time.Minute)

	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:15" })
}

func TestServiceRefusesAnInvalidReleveInterval(t *testing.T) {
	for _, bad := range []string{"souvent", "0s", "-1h"} {
		t.Run(bad, func(t *testing.T) {
			plakar := newFakePlakar(t)
			plakar.stores("")
			a, err := tryStartService(plakar, newClock("2026-11-06T17:00:05Z"), map[string]string{"PLAKAR_UI_INTERVALLE": bad})
			if err == nil {
				a.Close()
				t.Fatalf("le service a démarré avec l'intervalle %q", bad)
			}
			if !strings.Contains(err.Error(), "PLAKAR_UI_INTERVALLE") {
				t.Errorf("erreur = %q, doit nommer la variable", err)
			}
		})
	}
}

func TestRefreshButtonRelevesTheStoreAtOnce(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.ls("immich", response{Stdout: fixture(t, "ls-arr.txt")})
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, nil)
	svc.dashboard()
	clk.advance(5 * time.Minute)

	code, body := svc.releve("arr")

	if code != http.StatusOK {
		t.Fatalf("code %d après ↻\n%s", code, body)
	}
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 18:05" })
	if got := lsCalls(plakar, "arr"); got != 2 {
		t.Errorf("%d Relevés de arr, attendu 2", got)
	}
	// Les autres Stores ne sont pas relevés.
	_, rows := svc.dashboard()
	assertRow(t, rows["immich"], row{"releve": "2026-11-06 18:00"})
	if got := lsCalls(plakar, "immich"); got != 1 {
		t.Errorf("%d Relevés de immich, attendu 1", got)
	}
}

func TestReleveInProgressIsShownAndRefreshDoesNotStartASecondOne(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt"), DelayMS: 500})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.waitRow("arr", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })

	code, body := svc.releve("arr")

	if code != http.StatusOK {
		t.Fatalf("code %d après ↻\n%s", code, body)
	}
	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"releve-en-cours": "Relevé en cours"})
	got := svc.waitRow("arr", func(r row) bool { return r["releve-en-cours"] == "" })
	assertRow(t, got, row{"releve": "2026-11-06 18:00", "id": "93ebf1a2"})
	if got := lsCalls(plakar, "arr"); got != 1 {
		t.Errorf("%d Relevés de arr, attendu 1 : le ↻ pendant le Relevé en a lancé un second", got)
	}
}

func TestRefreshOfAnUnknownStoreIsNotFound(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	code, _ := svc.releve("inconnu")

	if code != http.StatusNotFound {
		t.Errorf("code %d, attendu %d", code, http.StatusNotFound)
	}
}

func TestRefreshIsRefusedFromAnotherSite(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()
	req, err := http.NewRequest(http.MethodPost, svc.app.URL()+"/stores/arr/releve", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("code %d, attendu %d", resp.StatusCode, http.StatusForbidden)
	}
	svc.dashboard()
	if got := lsCalls(plakar, "arr"); got != 1 {
		t.Errorf("%d Relevés de arr, attendu 1", got)
	}
}

func TestNoPlakarProcessStaysAliveBetweenTwoReleves(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"panne":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.ls("panne", response{Stderr: "plakar: could not resolve repository: @panne\n", Code: 1})
	clk := newClock("2026-11-06T17:00:05Z")
	svc := startService(t, plakar, clk, nil)
	svc.dashboard()
	clk.advance(time.Hour)
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 19:00" })
	svc.waitRow("panne", func(r row) bool { return r["releve"] == "2026-11-06 19:00" })

	pids := plakar.pids()
	if len(pids) != 5 { // store show, puis deux Relevés de chaque Store
		t.Fatalf("%d processus plakar lancés, attendu 5", len(pids))
	}
	for _, pid := range pids {
		// Le signal 0 ne fait que tester l'existence du processus.
		if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
			t.Errorf("le processus plakar %d est encore là (%v)", pid, err)
		}
	}
}

func TestPlannedReleveUpdatesTheStateFile(t *testing.T) {
	state := map[string]string{"PLAKAR_UI_STATE": filepath.Join(t.TempDir(), "state.json")}
	clk := newClock("2026-11-06T17:00:05Z")
	before := newFakePlakar(t)
	before.stores(`{"arr":{}}` + "\n")
	before.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	svc := startService(t, before, clk, state)
	svc.dashboard()
	clk.advance(time.Hour)
	svc.waitRow("arr", func(r row) bool { return r["releve"] == "2026-11-06 19:00" })
	svc.stop()

	after := newFakePlakar(t)
	after.stores(`{"arr":{}}` + "\n")
	after.ls("arr", response{DelayMS: 60_000})
	svc = startService(t, after, clk, state)

	_, rows := svc.dashboard()
	assertRow(t, rows["arr"], row{"releve": "2026-11-06 19:00", "releve-en-cours": "Relevé en cours", "id": "93ebf1a2"})
}

func TestReleveCutShortByAStopIsNotRecordedAsAFailure(t *testing.T) {
	state := map[string]string{"PLAKAR_UI_STATE": filepath.Join(t.TempDir(), "state.json")}
	clk := newClock("2026-11-06T17:00:05Z")
	first := newFakePlakar(t)
	first.stores(`{"arr":{}}` + "\n")
	first.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	svc := startService(t, first, clk, state)
	svc.dashboard()
	svc.stop()

	// Le service s'arrête pendant un Relevé.
	slow := newFakePlakar(t)
	slow.stores(`{"arr":{}}` + "\n")
	slow.ls("arr", response{DelayMS: 60_000})
	svc = startService(t, slow, clk, state)
	svc.waitRow("arr", func(r row) bool { return r["releve-en-cours"] == "Relevé en cours" })
	svc.stop()

	svc = startService(t, slow, clk, state)
	_, rows := svc.dashboard()
	assertRow(t, rows["arr"], row{"etat": "À jour", "error": "", "releve": "2026-11-06 18:00", "id": "93ebf1a2"})
}
