package app_test

// Harnais de test : démarre le service entier, l'interroge en HTTP, remplace
// plakar par un faux binaire qui rejoue des sorties enregistrées et note les
// arguments reçus, et injecte l'horloge.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/html"

	"plakar-ui/internal/app"
)

var fakePlakarBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fakeplakar-")
	if err != nil {
		panic(err)
	}
	fakePlakarBin = filepath.Join(dir, "fakeplakar")
	build := exec.Command("go", "build", "-o", fakePlakarBin, "plakar-ui/internal/fakeplakar")
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		panic("construction du faux plakar : " + err.Error())
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// response reprend le format lu par internal/fakeplakar.
type response struct {
	Args   []string `json:"args"`
	Stdout string   `json:"stdout"`
	Stderr string   `json:"stderr"`
	Code   int      `json:"code"`
}

// fakePlakar est un faux binaire plakar propre à un test.
type fakePlakar struct {
	t         *testing.T
	dir       string
	responses []response
}

func newFakePlakar(t *testing.T) *fakePlakar {
	t.Helper()
	return &fakePlakar{t: t, dir: t.TempDir()}
}

// on enregistre la réponse à rejouer pour ces arguments exacts.
func (f *fakePlakar) on(args []string, r response) {
	r.Args = args
	f.responses = append(f.responses, r)
}

// stores enregistre la sortie de `plakar store show -json`.
func (f *fakePlakar) stores(jsonl string) {
	f.on([]string{"store", "show", "-json"}, response{Stdout: jsonl})
}

// ls enregistre la sortie de `plakar at @<store> ls -tags`.
func (f *fakePlakar) ls(store string, r response) {
	f.on([]string{"at", "@" + store, "ls", "-tags"}, r)
}

// path écrit la table des réponses et renvoie le chemin d'un exécutable qui
// lance le faux plakar avec son propre dossier.
func (f *fakePlakar) path() string {
	f.t.Helper()
	raw, err := json.Marshal(f.responses)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, "responses.json"), raw, 0o600); err != nil {
		f.t.Fatal(err)
	}
	wrapper := filepath.Join(f.dir, "plakar")
	script := fmt.Sprintf("#!/bin/sh\nFAKE_PLAKAR_DIR=%q exec %q \"$@\"\n", f.dir, fakePlakarBin)
	if err := os.WriteFile(wrapper, []byte(script), 0o700); err != nil {
		f.t.Fatal(err)
	}
	return wrapper
}

// calls renvoie les arguments de chaque appel reçu, dans l'ordre.
func (f *fakePlakar) calls() [][]string {
	f.t.Helper()
	file, err := os.Open(filepath.Join(f.dir, "calls.jsonl"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		f.t.Fatal(err)
	}
	defer file.Close()
	var calls [][]string
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		var args []string
		if err := json.Unmarshal(sc.Bytes(), &args); err != nil {
			f.t.Fatal(err)
		}
		calls = append(calls, args)
	}
	return calls
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// clock est une horloge réglable par le test.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock(rfc3339 string) *clock {
	now, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		panic(err)
	}
	return &clock{now: now}
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// paris est le fuseau « heure locale » des tests.
var paris = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		panic(err)
	}
	return loc
}()

// service est le service démarré en entier pour un test.
type service struct {
	t   *testing.T
	app *app.App
}

// startService démarre le service avec ces variables d'environnement. Le
// chemin du faux plakar et une adresse libre sont fournis s'ils manquent.
func startService(t *testing.T, plakar *fakePlakar, clk *clock, env map[string]string) *service {
	t.Helper()
	a, err := tryStartService(plakar, clk, env)
	if err != nil {
		t.Fatalf("démarrage du service : %v", err)
	}
	t.Cleanup(func() { a.Close() })
	return &service{t: t, app: a}
}

// tryStartService démarre le service et renvoie son erreur de démarrage.
func tryStartService(plakar *fakePlakar, clk *clock, env map[string]string) (*app.App, error) {
	full := map[string]string{
		"PLAKAR_UI_ADDR":       "127.0.0.1:0",
		"PLAKAR_UI_PLAKAR_BIN": plakar.path(),
	}
	for k, v := range env {
		full[k] = v
	}
	return app.Start(context.Background(), app.Options{
		Getenv:   func(k string) string { return full[k] },
		Now:      clk.Now,
		Location: paris,
		Log:      io.Discard,
	})
}

func (s *service) get(path string) (int, string) {
	s.t.Helper()
	resp, err := http.Get(s.app.URL() + path)
	if err != nil {
		s.t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.t.Fatal(err)
	}
	return resp.StatusCode, string(body)
}

// row est le contenu affiché d'une ligne du tableau de bord : le texte de
// chaque cellule, par nom de champ.
type row map[string]string

// dashboard lit le tableau de bord une fois tous les Relevés de démarrage
// terminés, et renvoie ses lignes dans l'ordre d'affichage.
func (s *service) dashboard() (names []string, rows map[string]row) {
	s.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		code, body := s.get("/")
		if code != http.StatusOK {
			s.t.Fatalf("GET / : code %d\n%s", code, body)
		}
		names, rows, pending := parseDashboard(s.t, body)
		if !pending {
			return names, rows
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("Relevés toujours en attente :\n%s", body)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// parseDashboard lit les lignes <tr data-store="…">, et dans chacune les
// cellules <td data-field="…">.
func parseDashboard(t *testing.T, body string) (names []string, rows map[string]row, pending bool) {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	rows = map[string]row{}
	var walk func(n *html.Node, current row)
	walk = func(n *html.Node, current row) {
		if n.Type == html.ElementNode {
			if store, ok := attr(n, "data-store"); ok {
				current = row{}
				rows[store] = current
				names = append(names, store)
				if _, ok := attr(n, "data-pending"); ok {
					pending = true
				}
			}
			if field, ok := attr(n, "data-field"); ok && current != nil {
				current[field] = text(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, current)
		}
	}
	walk(doc, nil)
	return names, rows, pending
}

func attr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
