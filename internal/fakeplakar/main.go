// Command fakeplakar remplace le binaire plakar dans les tests.
//
// Il lit dans $FAKE_PLAKAR_DIR/responses.json une table de réponses
// enregistrées (arguments exacts → sortie standard, sortie d'erreur, code de
// retour), rejoue celle qui correspond aux arguments reçus, et ajoute chaque
// appel à $FAKE_PLAKAR_DIR/calls.jsonl (un tableau JSON d'arguments par ligne)
// et son PID à $FAKE_PLAKAR_DIR/pids, pour vérifier qu'aucun processus ne
// reste actif.
// Une réponse peut se faire attendre (DelayMS) pour imiter un
// Store lent, et remplacer d'autres réponses une fois rejouée (Replace) pour
// imiter une config plakar modifiée, par exemple par `store add`. Une réponse
// peut enfin imiter `plakar ui` (Serve) : comme lui, elle annonce sa sortie
// standard avant d'écouter sur l'adresse de l'option -addr (après ListenDelayMS),
// puis sert des pages jusqu'à son arrêt.
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Response est une sortie enregistrée de plakar pour des arguments donnés.
type Response struct {
	Args    []string `json:"args"`
	Stdout  string   `json:"stdout"`
	Stderr  string   `json:"stderr"`
	Code    int      `json:"code"`
	DelayMS int      `json:"delay_ms"`
	// Replace remplace, après cet appel, les réponses aux mêmes arguments, ou
	// les ajoute.
	Replace []Response `json:"replace,omitempty"`
	// Serve imite `plakar ui` : chaque page répond « <store> <méthode> <URI> »,
	// et 401 sans le paramètre plakar_token=<Token>.
	Serve bool   `json:"serve,omitempty"`
	Token string `json:"token,omitempty"`
	// ListenDelayMS sépare l'annonce de `plakar ui` de son écoute. Le vrai
	// plakar imprime sa ligne de lancement, puis seulement ouvre son port.
	ListenDelayMS int `json:"listen_delay_ms,omitempty"`
}

func main() {
	dir := os.Getenv("FAKE_PLAKAR_DIR")
	if dir == "" {
		fail("FAKE_PLAKAR_DIR non défini")
	}
	args := os.Args[1:]

	if err := record(filepath.Join(dir, "calls.jsonl"), args); err != nil {
		fail("journal des appels : %v", err)
	}
	if err := appendLine(filepath.Join(dir, "pids"), []byte(strconv.Itoa(os.Getpid()))); err != nil {
		fail("journal des PID : %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "responses.json"))
	if err != nil {
		fail("lecture des réponses : %v", err)
	}
	var responses []Response
	if err := json.Unmarshal(raw, &responses); err != nil {
		fail("réponses illisibles : %v", err)
	}
	for _, r := range responses {
		if slices.Equal(r.Args, args) {
			if len(r.Replace) > 0 {
				if err := replace(dir, responses, r.Replace); err != nil {
					fail("remplacement des réponses : %v", err)
				}
			}
			time.Sleep(time.Duration(r.DelayMS) * time.Millisecond)
			if r.Serve {
				serve(args, r)
			}
			fmt.Fprint(os.Stdout, r.Stdout)
			fmt.Fprint(os.Stderr, r.Stderr)
			os.Exit(r.Code)
		}
	}
	fail("aucune réponse enregistrée pour %q", args)
}

// replace réécrit la table des réponses, où news remplacent les réponses aux
// mêmes arguments. L'écriture est atomique : un appel simultané lit l'ancienne
// table ou la nouvelle.
func replace(dir string, responses, news []Response) error {
	for _, n := range news {
		responses = slices.DeleteFunc(responses, func(r Response) bool { return slices.Equal(r.Args, n.Args) })
		responses = append(responses, n)
	}
	raw, err := json.Marshal(responses)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".responses-*.json")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, "responses.json"))
}

// serve imite `plakar ui` sur l'adresse de l'option -addr, et ne rend jamais
// la main : le processus s'arrête sur un signal.
func serve(args []string, r Response) {
	i := slices.Index(args, "-addr")
	if i < 0 || i+1 >= len(args) {
		fail("serve : option -addr absente de %q", args)
	}
	store := strings.TrimPrefix(args[1], "@")
	fmt.Fprint(os.Stdout, r.Stdout)
	fmt.Fprint(os.Stderr, r.Stderr)
	time.Sleep(time.Duration(r.ListenDelayMS) * time.Millisecond)
	ln, err := net.Listen("tcp", args[i+1])
	if err != nil {
		fail("serve : %v", err)
	}
	err = http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("plakar_token") != r.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		fmt.Fprintf(w, "%s %s %s", store, req.Method, req.URL.RequestURI())
	}))
	fail("serve : %v", err)
}

func record(path string, args []string) error {
	line, err := json.Marshal(args)
	if err != nil {
		return err
	}
	return appendLine(path, line)
}

func appendLine(path string, line []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "fakeplakar: "+format+"\n", a...)
	os.Exit(99)
}
