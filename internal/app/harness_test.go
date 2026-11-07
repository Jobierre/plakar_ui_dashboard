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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	Args    []string `json:"args"`
	Stdout  string   `json:"stdout"`
	Stderr  string   `json:"stderr"`
	Code    int      `json:"code"`
	DelayMS int      `json:"delay_ms"`
	// Replace remplace, une fois cette réponse rejouée, les réponses aux mêmes
	// arguments (chacune porte ses Args).
	Replace []response `json:"replace,omitempty"`
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

// pids renvoie le PID de chaque processus plakar lancé.
func (f *fakePlakar) pids() []int {
	f.t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.dir, "pids"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		f.t.Fatal(err)
	}
	var pids []int
	for _, line := range strings.Fields(string(raw)) {
		pid, err := strconv.Atoi(line)
		if err != nil {
			f.t.Fatal(err)
		}
		pids = append(pids, pid)
	}
	return pids
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// clock est une horloge réglable par le test. Ses attentes (After) ne se
// terminent que quand le test fait passer le temps.
type clock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []waiter
}

// waiter est une attente en cours sur l'horloge.
type waiter struct {
	at time.Time
	ch chan time.Time
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

// After se termine quand l'horloge a avancé de d.
func (c *clock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	if d <= 0 {
		ch <- c.now
		return ch
	}
	c.waiters = append(c.waiters, waiter{at: c.now.Add(d), ch: ch})
	return ch
}

// advance fait passer le temps, et termine les attentes échues.
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	pending := c.waiters[:0]
	for _, w := range c.waiters {
		if w.at.After(c.now) {
			pending = append(pending, w)
			continue
		}
		w.ch <- c.now
	}
	c.waiters = pending
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
	// log reçoit les journaux du service.
	log *syncBuffer
}

// syncBuffer est un tampon sûr pour un accès concurrent.
type syncBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// startService démarre le service avec ces variables d'environnement. Le
// chemin du faux plakar, une adresse libre et un fichier d'état propre au test
// sont fournis s'ils manquent.
func startService(t *testing.T, plakar *fakePlakar, clk *clock, env map[string]string) *service {
	t.Helper()
	return startServiceWith(t, plakar.path(), clk, env)
}

// startServiceWith démarre le service sur ce binaire plakar.
func startServiceWith(t *testing.T, plakarBin string, clk *clock, env map[string]string) *service {
	t.Helper()
	if _, ok := env["PLAKAR_UI_STATE"]; !ok {
		env = withEnv(env, "PLAKAR_UI_STATE", filepath.Join(t.TempDir(), "state.json"))
	}
	log := &syncBuffer{}
	a, err := startApp(plakarBin, clk, env, log)
	if err != nil {
		t.Fatalf("démarrage du service : %v", err)
	}
	t.Cleanup(func() { a.Close() })
	return &service{t: t, app: a, log: log}
}

// withEnv renvoie une copie de env complétée de k=v.
func withEnv(env map[string]string, k, v string) map[string]string {
	full := map[string]string{k: v}
	for k, v := range env {
		full[k] = v
	}
	return full
}

// tryStartService démarre le service et renvoie son erreur de démarrage.
func tryStartService(plakar *fakePlakar, clk *clock, env map[string]string) (*app.App, error) {
	return tryStartServiceWith(plakar.path(), clk, env)
}

// tryStartServiceWith démarre le service sur ce binaire plakar et renvoie son
// erreur de démarrage.
func tryStartServiceWith(plakarBin string, clk *clock, env map[string]string) (*app.App, error) {
	return startApp(plakarBin, clk, env, io.Discard)
}

// startApp démarre le service sur ce binaire plakar, journaux vers log.
func startApp(plakarBin string, clk *clock, env map[string]string, log io.Writer) (*app.App, error) {
	full := map[string]string{
		"PLAKAR_UI_ADDR":         "127.0.0.1:0",
		"PLAKAR_UI_PLAKAR_BIN":   plakarBin,
		"PLAKAR_UI_UTILISATEURS": testUser,
	}
	for k, v := range env {
		full[k] = v
	}
	return app.Start(context.Background(), app.Options{
		Getenv:   func(k string) string { return full[k] },
		Now:      clk.Now,
		After:    clk.After,
		Location: paris,
		Log:      log,
	})
}

// testUser est l'Utilisateur autorisé par défaut des tests : le harnais pose
// son identité sur chaque requête, comme le ferait `tailscale serve`.
const testUser = "jobierre@example.com"

// identityHeader est l'en-tête d'identité posé par `tailscale serve`.
const identityHeader = "Tailscale-User-Login"

func (s *service) get(path string) (int, string) {
	s.t.Helper()
	req, err := http.NewRequest(http.MethodGet, s.app.URL()+path, nil)
	if err != nil {
		s.t.Fatal(err)
	}
	return s.do(req, testUser)
}

// request prépare une requête vers le service, à envoyer avec do.
func (s *service) request(method, path string) *http.Request {
	s.t.Helper()
	req, err := http.NewRequest(method, s.app.URL()+path, nil)
	if err != nil {
		s.t.Fatal(err)
	}
	return req
}

// do envoie la requête avec cette identité Tailscale (aucun en-tête si elle
// est vide), en suivant les redirections (qui gardent l'en-tête), et renvoie
// le code et le corps obtenus.
func (s *service) do(req *http.Request, user string) (int, string) {
	s.t.Helper()
	if user != "" {
		req.Header.Set(identityHeader, user)
	}
	resp, err := http.DefaultClient.Do(req)
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

// setSeuil envoie le formulaire de Seuil de retard d'un Store, comme le ferait
// le navigateur, et renvoie le code et la page obtenus.
func (s *service) setSeuil(store, seuil string) (int, string) {
	s.t.Helper()
	return s.postForm("/stores/"+url.PathEscape(store)+"/seuil", url.Values{"seuil": {seuil}})
}

// releve appuie sur le bouton ↻ d'un Store, comme le ferait le navigateur, et
// renvoie le code et la page obtenus.
func (s *service) releve(store string) (int, string) {
	s.t.Helper()
	return s.postForm("/stores/"+url.PathEscape(store)+"/releve", nil)
}

// postForm envoie un formulaire et renvoie le code et la page obtenus, après
// redirection.
func (s *service) postForm(path string, values url.Values) (int, string) {
	s.t.Helper()
	req, err := http.NewRequest(http.MethodPost, s.app.URL()+path, strings.NewReader(values.Encode()))
	if err != nil {
		s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return s.do(req, testUser)
}

// waitRow relit le tableau de bord jusqu'à ce que la ligne du Store vérifie
// ready, et la renvoie.
func (s *service) waitRow(store string, ready func(row) bool) row {
	s.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, body := s.get("/")
		_, rows, _ := parseDashboard(s.t, body)
		if r := rows[store]; r != nil && ready(r) {
			return r
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("ligne %q jamais prête :\n%s", store, body)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// stop arrête le service, comme un arrêt du conteneur.
func (s *service) stop() {
	s.t.Helper()
	if err := s.app.Close(); err != nil {
		s.t.Fatal(err)
	}
}

// row est le contenu affiché d'une ligne du tableau de bord : le texte de
// chaque cellule, par nom de champ, et son infobulle sous « <champ>.title ».
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
				if title, ok := attr(n, "title"); ok {
					current[field+".title"] = title
				}
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

// commande envoie une Commande de Store depuis son bloc, comme le ferait le
// navigateur, et renvoie le code et la page obtenus.
func (s *service) commande(line string) (int, string) {
	s.t.Helper()
	return s.postForm("/commande", url.Values{"commande": {line}})
}

// commandeBlock lit le résultat d'une Commande de Store dans la page : le
// texte brut de chaque élément <… data-commande="…">.
func commandeBlock(t *testing.T, body string) map[string]string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if field, ok := attr(n, "data-commande"); ok {
				got[field] = rawText(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return got
}

// rawText est le texte d'un nœud, blancs compris.
func rawText(n *html.Node) string {
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
	return b.String()
}
