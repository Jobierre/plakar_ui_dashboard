package commande

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Refus explique pourquoi une Commande de Store n'est pas exécutée.
type Refus struct{ msg string }

func (r *Refus) Error() string { return r.msg }

func refus(format string, a ...any) error { return &Refus{msg: fmt.Sprintf(format, a...)} }

// sousCommandes sont les seules sous-commandes de `plakar store` acceptées.
var sousCommandes = []string{"add", "set", "unset", "rm", "show"}

// usage rappelle ce qui est accepté.
var usage = "seules les commandes plakar store " + strings.Join(sousCommandes, " | ") + " sont acceptées"

// Parse découpe la ligne saisie en arguments et vérifie qu'elle est une
// Commande de Store acceptée. Il renvoie les arguments à passer au binaire
// plakar, sans le préfixe `plakar`, facultatif. Une erreur est un *Refus.
func Parse(line string) ([]string, error) {
	args, err := split(line)
	if err != nil {
		return nil, err
	}
	if len(args) > 0 && args[0] == "plakar" {
		args = args[1:]
	}
	switch {
	case len(args) == 0:
		return nil, refus("Commande vide : %s.", usage)
	case args[0] != "store":
		return nil, refus("Commande refusée : %s.", usage)
	case len(args) == 1:
		return nil, refus("Sous-commande manquante : %s.", usage)
	case !slices.Contains(sousCommandes, args[1]):
		return nil, refus("plakar store %s est refusé : %s.", args[1], usage)
	}
	// plakar lit ses options avec le paquet flag de Go, qui accepte -secrets,
	// --secrets et -secrets=<valeur>. Elles sont refusées à toute position,
	// et sans égard à la casse, par prudence.
	for _, a := range args[2:] {
		name, _, _ := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if strings.HasPrefix(a, "-") && strings.EqualFold(name, "secrets") {
			return nil, refus("L'option -secrets est refusée : elle afficherait les passphrases. Sans elle, plakar store show masque les secrets.")
		}
	}
	return args, nil
}

// errQuote signale un guillemet non fermé.
var errQuote = errors.New("guillemet non fermé")

// split découpe une ligne en arguments comme un shell POSIX, mais sans aucune
// expansion : les blancs séparent les arguments ; entre guillemets simples,
// tout est littéral ; entre guillemets doubles, seuls \" et \\ sont des
// échappements ; hors guillemets, \ rend littéral le caractère suivant. Les
// caractères ; | & $ ` ( ) n'ont aucun sens particulier.
func split(line string) ([]string, error) {
	var (
		args []string
		cur  strings.Builder
		// inArg distingue un argument vide entre guillemets ("") d'une
		// absence d'argument.
		inArg bool
	)
	rs := []rune(line)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		case r == '\'':
			inArg = true
			end := slices.Index(rs[i+1:], '\'')
			if end < 0 {
				return nil, refus("Commande refusée : %s.", errQuote)
			}
			cur.WriteString(string(rs[i+1 : i+1+end]))
			i += end + 1
		case r == '"':
			inArg = true
			closed := false
			for i++; i < len(rs); i++ {
				if rs[i] == '"' {
					closed = true
					break
				}
				if rs[i] == '\\' && i+1 < len(rs) && (rs[i+1] == '"' || rs[i+1] == '\\') {
					i++
				}
				cur.WriteRune(rs[i])
			}
			if !closed {
				return nil, refus("Commande refusée : %s.", errQuote)
			}
		case r == '\\':
			inArg = true
			if i+1 < len(rs) {
				i++
				cur.WriteRune(rs[i])
			}
		default:
			inArg = true
			cur.WriteRune(r)
		}
	}
	if inArg {
		args = append(args, cur.String())
	}
	return args, nil
}
