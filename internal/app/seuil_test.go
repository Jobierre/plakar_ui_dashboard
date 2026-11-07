package app_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// arrSixHoursAgo démarre le service sur un Store « arr » dont le Dernier
// snapshot date de 6 h.
func arrSixHoursAgo(t *testing.T) *service {
	t.Helper()
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.ls("immich", response{Stdout: fixture(t, "ls-arr.txt")})
	return startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
}

func TestSeuilOfOneStoreIsChangedFromTheDashboardAndEtatFollowsAtOnce(t *testing.T) {
	svc := arrSixHoursAgo(t)
	svc.dashboard()

	code, body := svc.setSeuil("arr", "4h")

	if code != http.StatusOK {
		t.Fatalf("code %d après le changement de Seuil\n%s", code, body)
	}
	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"seuil": "4h", "etat": "En retard", "seuil-error": ""})
	// Les autres Stores gardent le Seuil par défaut.
	assertRow(t, rows["immich"], row{"seuil": "26h (par défaut)", "etat": "À jour"})
}

func TestSeuilAcceptsDaysAndAnEmptyValueRestoresTheDefault(t *testing.T) {
	svc := arrSixHoursAgo(t)
	svc.dashboard()

	_, body := svc.setSeuil("arr", "8d")
	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"seuil": "8d", "etat": "À jour"})

	_, body = svc.setSeuil("arr", "1d12h")
	_, rows, _ = parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"seuil": "36h"})

	_, body = svc.setSeuil("arr", "")
	_, rows, _ = parseDashboard(t, body)
	assertRow(t, rows["arr"], row{"seuil": "26h (par défaut)"})
}

func TestInvalidSeuilIsRefusedWithAMessageAndChangesNothing(t *testing.T) {
	for _, bad := range []string{"demain", "-4h", "0h", "8 jours", "1d-2h", "999999999d", "106000d100000h"} {
		t.Run(bad, func(t *testing.T) {
			svc := arrSixHoursAgo(t)
			svc.dashboard()
			svc.setSeuil("arr", "8d")

			code, body := svc.setSeuil("arr", bad)

			if code != http.StatusBadRequest {
				t.Errorf("code %d, attendu %d", code, http.StatusBadRequest)
			}
			_, rows, _ := parseDashboard(t, body)
			if rows["arr"]["seuil-error"] == "" {
				t.Errorf("aucun message pour le Seuil %q : %v", bad, rows["arr"])
			}
			_, rows = svc.dashboard()
			assertRow(t, rows["arr"], row{"seuil": "8d", "seuil-error": ""})
		})
	}
}

func TestSeuilOfAnUnknownStoreIsNotFound(t *testing.T) {
	svc := arrSixHoursAgo(t)

	code, _ := svc.setSeuil("inconnu", "4h")

	if code != http.StatusNotFound {
		t.Errorf("code %d, attendu %d", code, http.StatusNotFound)
	}
}

func TestSeuilFormIsRefusedFromAnotherSite(t *testing.T) {
	svc := arrSixHoursAgo(t)
	svc.dashboard()
	// Ce qu'envoie un navigateur quand une page tierce poste le formulaire.
	req, err := http.NewRequest(http.MethodPost, svc.app.URL()+"/stores/arr/seuil",
		strings.NewReader(url.Values{"seuil": {"1000d"}}.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	// Un Utilisateur autorisé : seule l'origine de la requête la fait refuser.
	code, _ := svc.do(req, testUser)

	if code != http.StatusForbidden {
		t.Errorf("code %d, attendu %d", code, http.StatusForbidden)
	}
	_, rows := svc.dashboard()
	assertRow(t, rows["arr"], row{"seuil": "26h (par défaut)"})
}
