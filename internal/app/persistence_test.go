package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAfterARestartTheDashboardShowsTheLastKnownValuesAndSeuilsBeforeANewReleve(t *testing.T) {
	state := map[string]string{"PLAKAR_UI_STATE": filepath.Join(t.TempDir(), "appdata", "state.json")}
	clk := newClock("2026-11-06T17:00:05Z")
	before := newFakePlakar(t)
	before.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	before.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	before.ls("immich", response{Stdout: fixture(t, "ls-arr.txt")})
	svc := startService(t, before, clk, state)
	svc.dashboard()
	svc.setSeuil("arr", "4h")
	svc.stop()

	// Au redémarrage, les Stores ne répondent pas avant longtemps.
	after := newFakePlakar(t)
	after.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	after.ls("arr", response{Stdout: "", DelayMS: 60_000})
	after.ls("immich", response{Stdout: "", DelayMS: 60_000})
	svc = startService(t, after, clk, state)

	_, rows := svc.dashboard()

	assertRow(t, rows["arr"], row{
		"etat":           "En retard",
		"seuil":          "4h",
		"id":             "93ebf1a2",
		"size":           "1.3 GiB",
		"size-ecart":     "0 %",
		"duration-ecart": "−4s",
		"tags":           "arr",
	})
	assertRow(t, rows["immich"], row{"etat": "À jour", "seuil": "26h (par défaut)", "id": "93ebf1a2"})
}

func TestInjoignableStoreKeepsItsLastKnownValuesWithTheError(t *testing.T) {
	state := map[string]string{"PLAKAR_UI_STATE": filepath.Join(t.TempDir(), "state.json")}
	clk := newClock("2026-11-06T17:00:05Z")
	before := newFakePlakar(t)
	before.stores(`{"arr":{}}` + "\n")
	before.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	svc := startService(t, before, clk, state)
	svc.dashboard()
	svc.stop()

	after := newFakePlakar(t)
	after.stores(`{"arr":{}}` + "\n")
	after.ls("arr", response{Stderr: "plakar: failed to read header: connection reset by peer\n", Code: 1})
	svc = startService(t, after, clk, state)

	got := svc.waitRow("arr", func(r row) bool { return r["error"] != "" })

	assertRow(t, got, row{
		"etat":  "Injoignable",
		"error": "plakar: failed to read header: connection reset by peer",
		"id":    "93ebf1a2",
		"date":  "2026-11-06 12:00",
		"size":  "1.3 GiB",
	})
}

func TestServiceRefusesToStartOnAnUnreadableStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{pas du json"), 0o600); err != nil {
		t.Fatal(err)
	}
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	a, err := tryStartService(plakar, newClock("2026-11-06T17:00:05Z"), map[string]string{"PLAKAR_UI_STATE": path})
	if err == nil {
		a.Close()
		t.Fatal("le service a démarré sur un fichier d'état illisible, au risque d'écraser les Seuils")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("erreur = %q, doit nommer le fichier d'état", err)
	}
}
