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
	plakar := newFakePlakar(t)
	plakar.stores("")
	a, err := tryStartService(plakar, newClock("2026-11-06T17:00:00Z"),
		map[string]string{"PLAKAR_UI_ADDR": "0.0.0.0:8080"})
	if err == nil {
		a.Close()
		t.Fatal("le service a accepté d'écouter sur 0.0.0.0")
	}
	if !strings.Contains(err.Error(), "localhost") {
		t.Errorf("erreur = %q, doit expliquer que l'adresse doit être sur localhost", err)
	}
}
