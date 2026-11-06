package app_test

import (
	"slices"
	"strings"
	"testing"
)

func TestDashboardShowsOneRowPerStoreOfThePlakarConfigInNameOrder(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(fixture(t, "store-show.jsonl"))
	for _, s := range []string{"arr", "immich", "vide", "panne", "petit"} {
		plakar.ls(s, response{})
	}
	svc := startService(t, plakar, newClock("2026-11-06T17:00:00Z"), nil)

	names, _ := svc.dashboard()

	// plakar rend ses Stores dans un ordre aléatoire : l'affichage est trié.
	want := []string{"arr", "immich", "panne", "petit", "vide"}
	if !slices.Equal(names, want) {
		t.Fatalf("Stores affichés = %v, attendu %v", names, want)
	}
}

func TestRowShowsTheLatestSnapshotOfTheStore(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{"location":"http://unraid:9876","passphrase":"********"}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	_, rows := svc.dashboard()

	assertRow(t, rows["arr"], row{
		"name":     "arr",
		"date":     "2026-11-06 12:00", // 11:00:05Z en heure de Paris
		"age":      "il y a 6 h",
		"id":       "93ebf1a2",
		"size":     "1.3 GiB",
		"duration": "21s",
		"path":     "/root/docker",
		"tags":     "arr",
	})
}

// assertRow vérifie les champs attendus d'une ligne, et seulement ceux-là.
func assertRow(t *testing.T, got, want row) {
	t.Helper()
	if got == nil {
		t.Fatalf("ligne absente, attendu %v", want)
	}
	for field, v := range want {
		if got[field] != v {
			t.Errorf("champ %q = %q, attendu %q", field, got[field], v)
		}
	}
}

func TestRowsReadSizesInEveryUnitTagsOrNoneAndEmptyStores(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"kib":{}}` + "\n" + `{"octets":{}}` + "\n" + `{"mib":{}}` + "\n" + `{"vide":{}}` + "\n")
	// Sorties réelles d'un Kloset jetable (plakar v1.1.6).
	small := strings.SplitAfter(fixture(t, "ls-small.txt"), "\n")
	plakar.ls("kib", response{Stdout: small[0] + small[1]})
	plakar.ls("octets", response{Stdout: small[1]})
	plakar.ls("mib", response{Stdout: "2026-11-05T22:00:12Z   0c1d2e3f   812.4 MiB     1m11s /mnt/user/immich tags=immich\n"})
	plakar.ls("vide", response{Stdout: ""})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:00Z"), nil)

	_, rows := svc.dashboard()

	assertRow(t, rows["kib"], row{"size": "4.9 KiB", "path": "/data/mes documents", "tags": "foo, bar", "id": "19ae0c83"})
	assertRow(t, rows["octets"], row{"size": "3 B", "path": "/data/mes documents", "tags": "", "id": "e2b9b87d"})
	assertRow(t, rows["mib"], row{"size": "812.4 MiB", "duration": "1m11s", "age": "il y a 18 h"})
	assertRow(t, rows["vide"], row{"name": "vide", "latest": "Aucun snapshot"})
	if _, ok := rows["vide"]["id"]; ok {
		t.Errorf("un Store vide ne doit pas afficher d'ID : %v", rows["vide"])
	}
}

func TestFailedReleveShowsItsErrorWithoutHidingOtherStores(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"panne":{}}` + "\n" + `{"format":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	// Sortie réelle de plakar v1.1.6 pour un Store qu'il ne sait pas ouvrir.
	plakar.ls("panne", response{Stderr: "plakar: could not resolve repository: @panne\n", Code: 1})
	plakar.ls("format", response{Stdout: "un format que plakar n'a jamais eu\n"})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	_, rows := svc.dashboard()

	assertRow(t, rows["arr"], row{"id": "93ebf1a2"})
	assertRow(t, rows["panne"], row{"error": "plakar: could not resolve repository: @panne"})
	if got := rows["format"]["error"]; !strings.Contains(got, "un format que plakar n'a jamais eu") {
		t.Errorf("erreur de lecture affichée = %q, doit citer la ligne illisible", got)
	}
}
