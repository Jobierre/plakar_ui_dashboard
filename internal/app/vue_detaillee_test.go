package app_test

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// Vue détaillée : une ligne compacte par Store, et un dépliage pour le reste.

func TestRowAndDepliageShareTheFieldsOfTheStore(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2   40.2 MiB       21s /root/docker tags=arr\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad    1.3 GiB       25s /root/docker tags=arr\n"})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()

	_, body := svc.get("/")

	arr := storeParts(t, body)["arr"]
	assertFieldsIn(t, "ligne", arr.ligne, "voyant", "etat", "name", "age", "size", "size-ecart", "releve-heure")
	assertFieldsIn(t, "dépliage", arr.depliage, "date", "id", "duration", "duration-ecart", "path", "tags", "releve", "seuil", "seuil-error")
	assertFieldsNotIn(t, "ligne", arr.ligne, "date", "id", "duration", "path", "tags", "seuil")
	assertFieldsNotIn(t, "dépliage", arr.depliage, "etat", "age", "size", "releve-heure")
	assertRow(t, arr.ligne, row{"releve-heure": "18:00"})
	assertRow(t, arr.depliage, row{"releve": "2026-11-06 18:00"})
	if arr.open {
		t.Errorf("le dépliage de arr doit être fermé par défaut")
	}
}

func TestEcartHasAnArrowASignAndThePreviousSnapshotAsTooltip(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"baisse":{}}` + "\n" + `{"hausse":{}}` + "\n" + `{"stable":{}}` + "\n" + `{"depuis-zero":{}}` + "\n")
	plakar.ls("baisse", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2   40.2 MiB       21s /root/docker\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad    1.3 GiB       25s /root/docker\n"})
	plakar.ls("hausse", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2   1.5 KiB     1m11s /data\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad   1.0 KiB       25s /data\n"})
	plakar.ls("stable", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2   1.3 GiB       25s /data\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad   1.3 GiB       25s /data\n"})
	plakar.ls("depuis-zero", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2      3 B        0s /data\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad      0 B        0s /data\n"})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()

	_, body := svc.get("/")

	stores := storeParts(t, body)
	assertRow(t, stores["baisse"].ligne, row{"size-ecart-sens": "↓", "size-ecart": "−97 %"})
	assertRow(t, stores["baisse"].depliage, row{"duration-ecart-sens": "↓", "duration-ecart": "−4s"})
	assertRow(t, stores["hausse"].ligne, row{"size-ecart-sens": "↑", "size-ecart": "+50 %"})
	assertRow(t, stores["hausse"].depliage, row{"duration-ecart-sens": "↑"})
	assertRow(t, stores["stable"].ligne, row{"size-ecart-sens": "→", "size-ecart": "0 %"})
	assertRow(t, stores["stable"].depliage, row{"duration-ecart-sens": "→"})
	assertRow(t, stores["depuis-zero"].ligne, row{"size-ecart-sens": "↑", "size-ecart": "nouveau"})
	_, rows, _ := parseDashboard(t, body)
	assertRow(t, rows["baisse"], row{"size-ecart.title": "par rapport au Snapshot précédent a4e9d6ad (1.3 GiB, 25s)"})
}

func TestInjoignableStoreShowsItsErrorInTheRowAndDimsItsLastKnownValues(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"panne":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	// Le premier Relevé de panne réussit, le suivant échoue.
	plakar.ls("panne", response{Stdout: fixture(t, "ls-arr.txt"), Replace: []response{
		{Args: []string{"at", "@panne", "ls", "-tags"}, Stderr: "plakar: failed to read header: connection reset by peer\n", Code: 1},
	}})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()
	svc.releve("panne")
	svc.waitRow("panne", func(r row) bool { return r["error"] != "" })

	_, body := svc.get("/")

	stores := storeParts(t, body)
	panne := stores["panne"]
	assertRow(t, panne.ligne, row{"etat": "Injoignable", "error": "plakar: failed to read header: connection reset by peer", "age": "il y a 6 h"})
	if !panne.perime {
		t.Errorf("les dernières valeurs connues de panne doivent être atténuées")
	}
	if stores["arr"].perime {
		t.Errorf("les valeurs de arr, À jour, ne doivent pas être atténuées")
	}
}

func TestDepliageOpensByItselfAfterARefusedSeuil(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n" + `{"immich":{}}` + "\n")
	plakar.ls("arr", response{Stdout: fixture(t, "ls-arr.txt")})
	plakar.ls("immich", response{Stdout: fixture(t, "ls-arr.txt")})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()

	code, body := svc.setSeuil("arr", "bientôt")

	if code != http.StatusBadRequest {
		t.Errorf("code %d, attendu %d", code, http.StatusBadRequest)
	}
	stores := storeParts(t, body)
	if !stores["arr"].open || stores["arr"].depliage["seuil-error"] == "" {
		t.Errorf("le dépliage de arr doit s'ouvrir sur l'erreur de Seuil : %+v", stores["arr"])
	}
	if stores["immich"].open {
		t.Errorf("le dépliage de immich doit rester fermé")
	}
}

func TestDepliageOpensByItselfAfterAFailedExploration(t *testing.T) {
	base := freePorts(3)
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	plakar.ls("arr", response{})
	plakar.on(uiArgs("arr", base), response{Stderr: "plakar: failed to open repository: bad passphrase\n", Code: 1})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), explorationEnv(base, nil))
	svc.dashboard()

	_, body := svc.explorer("arr")

	arr := storeParts(t, body)["arr"]
	if !arr.open {
		t.Errorf("le dépliage de arr doit s'ouvrir sur l'échec de l'Exploration")
	}
	assertRow(t, arr.depliage, row{"exploration-error": "plakar: failed to open repository: bad passphrase"})
}

func TestGererLesStoresIsClosedAndOpensOnACommandeResult(t *testing.T) {
	svc, _ := oneStore(t, nil)

	_, body := svc.get("/")
	if open, ok := gererOpen(t, body); !ok || open {
		t.Errorf("section « Gérer les Stores » : présente %v, ouverte %v ; attendu présente et fermée", ok, open)
	}

	_, body = svc.commande("plakar sync")
	if open, _ := gererOpen(t, body); !open {
		t.Errorf("la section « Gérer les Stores » doit s'ouvrir sur le refus de la commande")
	}
	if commandeBlock(t, body)["refus"] == "" {
		t.Errorf("refus attendu dans la section")
	}
}

func TestEachStoreAppearsOnceInThePage(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(fixture(t, "store-show.jsonl"))
	for _, s := range []string{"arr", "immich", "vide", "panne", "petit"} {
		plakar.ls(s, response{Stdout: fixture(t, "ls-arr.txt")})
	}
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)
	svc.dashboard()

	_, body := svc.get("/")

	seen := map[string]int{}
	fields := map[string]int{}
	var walk func(n *html.Node, store string)
	walk = func(n *html.Node, store string) {
		if n.Type == html.ElementNode {
			if s, ok := attr(n, "data-store"); ok {
				seen[s]++
				store = s
			}
			if f, ok := attr(n, "data-field"); ok && store != "" {
				fields[store+" "+f]++
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, store)
		}
	}
	walk(parseHTML(t, body), "")
	if len(seen) != 5 {
		t.Errorf("%d Stores dans la page, attendu 5 : %v", len(seen), seen)
	}
	for s, n := range seen {
		if n != 1 {
			t.Errorf("Store %q présent %d fois", s, n)
		}
	}
	for f, n := range fields {
		if n != 1 {
			t.Errorf("champ %q présent %d fois", f, n)
		}
	}
}

// gererOpen dit si la section « Gérer les Stores » (<details data-gerer>) est
// dans la page, et si elle est ouverte.
func gererOpen(t *testing.T, body string) (open, ok bool) {
	t.Helper()
	el := findElement(t, body, func(n *html.Node) bool { _, ok := attr(n, "data-gerer"); return ok })
	if el == nil {
		return false, false
	}
	_, open = attr(el, "open")
	return open, true
}

// parts est un Store de la vue détaillée : les champs de sa ligne repliée, ceux
// de son dépliage, et si le dépliage est ouvert.
type parts struct {
	ligne, depliage row
	open            bool
	// perime : le Store atténue ses valeurs, les dernières connues.
	perime bool
}

// storeParts lit, pour chaque <… data-store="…">, les champs de sa ligne
// (<… data-ligne>) et de son dépliage (<… data-depliage>).
func storeParts(t *testing.T, body string) map[string]*parts {
	t.Helper()
	doc := parseHTML(t, body)
	out := map[string]*parts{}
	var walk func(n *html.Node, p *parts, part row)
	walk = func(n *html.Node, p *parts, part row) {
		if n.Type == html.ElementNode {
			if store, ok := attr(n, "data-store"); ok {
				_, open := attr(n, "open")
				class, _ := attr(n, "class")
				p = &parts{ligne: row{}, depliage: row{}, open: open, perime: strings.Contains(" "+class+" ", " perime ")}
				out[store] = p
				part = nil
			}
			if _, ok := attr(n, "data-ligne"); ok && p != nil {
				part = p.ligne
			}
			if _, ok := attr(n, "data-depliage"); ok && p != nil {
				part = p.depliage
			}
			if field, ok := attr(n, "data-field"); ok && part != nil {
				part[field] = text(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, p, part)
		}
	}
	walk(doc, nil, nil)
	return out
}

func parseHTML(t *testing.T, body string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func assertFieldsIn(t *testing.T, where string, got row, fields ...string) {
	t.Helper()
	for _, f := range fields {
		if _, ok := got[f]; !ok {
			t.Errorf("champ %q absent de la %s : %v", f, where, got)
		}
	}
}

func assertFieldsNotIn(t *testing.T, where string, got row, fields ...string) {
	t.Helper()
	for _, f := range fields {
		if _, ok := got[f]; ok {
			t.Errorf("champ %q présent dans la %s, attendu ailleurs", f, where)
		}
	}
}
