package app_test

import (
	"slices"
	"strings"
	"testing"
)

func TestStartupListsStoresThenMakesOneReleveOfEachWithTheConfiguredPlakar(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	plakar.ls("arr", response{})
	plakar.ls("immich", response{})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:00Z"), nil)

	svc.dashboard()

	calls := plakar.calls()
	if len(calls) == 0 || !slices.Equal(calls[0], []string{"store", "show", "-json"}) {
		t.Fatalf("premier appel = %v, attendu la liste des Stores", calls)
	}
	var releves []string
	for _, c := range calls[1:] {
		releves = append(releves, strings.Join(c, " "))
	}
	slices.Sort(releves)
	want := []string{"at @arr ls -tags", "at @immich ls -tags"}
	if !slices.Equal(releves, want) {
		t.Fatalf("Relevés = %q, attendu %q", releves, want)
	}
}

func TestServiceRefusesToListenOutsideLocalhost(t *testing.T) {
	// Toutes les formes d'écoute sur toutes les interfaces, et une adresse
	// du LAN.
	for _, addr := range []string{"0.0.0.0:8080", ":8080", "[::]:8080", "192.168.1.10:8080"} {
		plakar := newFakePlakar(t)
		plakar.stores("")
		a, err := tryStartService(plakar, newClock("2026-11-06T17:00:00Z"),
			map[string]string{"PLAKAR_UI_ADDR": addr})
		if err == nil {
			a.Close()
			t.Errorf("le service a accepté d'écouter sur %q", addr)
			continue
		}
		if !strings.Contains(err.Error(), "localhost") {
			t.Errorf("%q : erreur = %q, doit expliquer que l'adresse doit être sur localhost", addr, err)
		}
	}
}
