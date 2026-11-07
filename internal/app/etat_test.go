package app_test

import (
	"testing"
	"time"
)

func TestEtatComparesTheLatestSnapshotToTheDefaultSeuilOf26h(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"recent":{}}` + "\n" + `{"ancien":{}}` + "\n" + `{"vide":{}}` + "\n")
	plakar.ls("recent", response{Stdout: "2026-11-06T11:00:05Z   93ebf1a2   1.3 GiB       21s /root/docker\n"})
	plakar.ls("ancien", response{Stdout: "2026-11-05T14:00:00Z   a4e9d6ad   1.3 GiB       21s /root/docker\n"})
	plakar.ls("vide", response{Stdout: ""})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	_, rows := svc.dashboard()

	assertRow(t, rows["recent"], row{"etat": "À jour", "seuil": "26h (par défaut)"}) // 6 h
	assertRow(t, rows["ancien"], row{"etat": "En retard"})                           // 27 h
	// Sans aucun snapshot, aucun Dernier snapshot n'est dans le Seuil.
	assertRow(t, rows["vide"], row{"etat": "En retard"})
}

func TestStoreFallsBehindByTheMerePassingOfTimeWithoutANewReleve(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")}) // 2026-11-06T11:00:05Z
	clk := newClock("2026-11-06T17:00:05Z")
	// Aucun Relevé planifié ne tombe pendant le test.
	svc := startService(t, plakar, clk, map[string]string{"PLAKAR_UI_INTERVALLE": "1000h"})
	_, rows := svc.dashboard()
	assertRow(t, rows["arr"], row{"etat": "À jour"})
	calls := len(plakar.calls())

	clk.advance(20*time.Hour + time.Second) // 26 h et 1 s

	_, rows = svc.dashboard()
	assertRow(t, rows["arr"], row{"etat": "En retard", "age": "il y a 26 h"})
	if got := len(plakar.calls()); got != calls {
		t.Errorf("%d appels à plakar après le passage du temps, attendu %d", got, calls)
	}
}

func TestDefaultSeuilIsSetByEnvironment(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")}) // il y a 6 h
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"),
		map[string]string{"PLAKAR_UI_SEUIL": "4h"})

	_, rows := svc.dashboard()

	assertRow(t, rows["arr"], row{"etat": "En retard", "seuil": "4h (par défaut)"})
}

func TestServiceRefusesAnInvalidDefaultSeuil(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores("")
	a, err := tryStartService(plakar, newClock("2026-11-06T17:00:00Z"),
		map[string]string{"PLAKAR_UI_SEUIL": "demain"})
	if err == nil {
		a.Close()
		t.Fatal("le service a accepté un Seuil de retard « demain »")
	}
}

func TestStoreWhoseReleveFailedIsInjoignableWithItsError(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"panne":{}}` + "\n")
	plakar.ls("panne", response{Stderr: "plakar: could not resolve repository: @panne\n", Code: 1})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	_, rows := svc.dashboard()

	assertRow(t, rows["panne"], row{"etat": "Injoignable", "error": "plakar: could not resolve repository: @panne"})
}

func TestEcartNeverChangesTheEtat(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"vidé":{}}` + "\n")
	// La source s'est vidée, mais le Dernier snapshot est dans le Seuil.
	plakar.ls("vidé", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2   40.2 MiB       21s /root/docker\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad    1.3 GiB       25s /root/docker\n"})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	_, rows := svc.dashboard()

	assertRow(t, rows["vidé"], row{"size-ecart": "−97 %", "etat": "À jour"})
}
