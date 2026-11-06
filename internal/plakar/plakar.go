// Package plakar pilote le binaire plakar en sous-processus.
//
// Le binaire est toujours appelé directement, sans shell : aucun argument
// n'est jamais interprété.
package plakar

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// Plakar désigne le binaire plakar à appeler.
type Plakar struct {
	Bin string
}

// Result est le résultat d'un appel à plakar.
type Result struct {
	Stdout string
	Stderr string
	Code   int
}

// Run lance plakar avec ces arguments et attend sa fin. Une erreur n'est
// renvoyée que si le processus n'a pas pu être lancé ou a été interrompu ; un
// code de retour non nul est rapporté dans Result.
func (p Plakar) Run(ctx context.Context, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, p.Bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	var exit *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exit) && ctx.Err() == nil:
		res.Code = exit.ExitCode()
	default:
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		return res, err
	}
	return res, nil
}

// Stores renvoie les noms des Stores de la config plakar, triés.
//
// plakar v1.1.6 n'a pas de `store ls` : on lit `store show -json`, qui émet
// une ligne {"<nom>": {…}} par Store avec les secrets masqués, et on n'en
// garde que les noms.
func (p Plakar) Stores(ctx context.Context) ([]string, error) {
	out, err := p.Output(ctx, "store", "show", "-json")
	if err != nil {
		return nil, err
	}
	var names []string
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("store show : ligne illisible : %w", err)
		}
		for name := range entry {
			names = append(names, name)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

// Output lance plakar et renvoie sa sortie standard. Un code de retour non nul
// est une erreur, décrite par la sortie d'erreur de plakar.
func (p Plakar) Output(ctx context.Context, args ...string) (string, error) {
	res, err := p.Run(ctx, args...)
	if err != nil {
		return "", err
	}
	if res.Code != 0 {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = fmt.Sprintf("code de retour %d", res.Code)
		}
		return "", errors.New(msg)
	}
	return res.Stdout, nil
}
