// Command fakeplakar remplace le binaire plakar dans les tests.
//
// Il lit dans $FAKE_PLAKAR_DIR/responses.json une table de réponses
// enregistrées (arguments exacts → sortie standard, sortie d'erreur, code de
// retour), rejoue celle qui correspond aux arguments reçus, et ajoute chaque
// appel à $FAKE_PLAKAR_DIR/calls.jsonl (un tableau JSON d'arguments par ligne)
// et son PID à $FAKE_PLAKAR_DIR/pids, pour vérifier qu'aucun processus ne
// reste actif.
// Une réponse peut se faire attendre (DelayMS) pour imiter un
// Store lent.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"
)

// Response est une sortie enregistrée de plakar pour des arguments donnés.
type Response struct {
	Args    []string `json:"args"`
	Stdout  string   `json:"stdout"`
	Stderr  string   `json:"stderr"`
	Code    int      `json:"code"`
	DelayMS int      `json:"delay_ms"`
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
			time.Sleep(time.Duration(r.DelayMS) * time.Millisecond)
			fmt.Fprint(os.Stdout, r.Stdout)
			fmt.Fprint(os.Stderr, r.Stderr)
			os.Exit(r.Code)
		}
	}
	fail("aucune réponse enregistrée pour %q", args)
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
