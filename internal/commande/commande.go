// Package commande exécute les Commandes de Store : des commandes
// `plakar store …` saisies dans le tableau de bord.
//
// Une Commande de Store n'est jamais journalisée ni conservée : elle peut
// contenir une passphrase.
package commande

import (
	"context"
	"fmt"
	"sync"
	"time"

	"plakar-ui/internal/plakar"
)

// timeout borne la durée d'une Commande de Store.
const timeout = 2 * time.Minute

// Executeur exécute les Commandes de Store.
type Executeur struct {
	Plakar plakar.Plakar
	// Rafraichir relit la config plakar dans le Catalogue. Il est appelé après
	// une Commande réussie (code de retour 0), qui a pu la modifier.
	Rafraichir func(ctx context.Context) error

	// mu sérialise les Commandes : chacune rafraîchit le Catalogue d'après
	// la config plakar qu'elle vient de modifier.
	mu sync.Mutex
}

// Sortie est ce que plakar a rendu pour une Commande de Store.
type Sortie struct {
	Stdout, Stderr string
	Code           int
}

// Run exécute la ligne saisie si c'est une Commande de Store acceptée, en
// appelant directement le binaire plakar, sans shell. Un refus est un *Refus,
// et plakar n'est alors pas appelé. Une Sortie accompagnée d'une erreur
// signale une Commande réussie dont le rafraîchissement a échoué.
//
// Une fois lancée, la Commande va à son terme, rafraîchissement compris, même
// si ctx (la requête du navigateur) est annulé : sinon la config plakar
// pourrait changer sans que le Catalogue le sache.
func (c *Executeur) Run(ctx context.Context, line string) (*Sortie, error) {
	args, err := Parse(line)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, done := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer done()
	res, err := c.Plakar.Run(ctx, args...)
	if err != nil {
		// L'erreur ne reprend pas les arguments : ils peuvent contenir une
		// passphrase.
		return nil, fmt.Errorf("plakar n'a pas pu être exécuté : %w", err)
	}
	sortie := &Sortie{Stdout: res.Stdout, Stderr: res.Stderr, Code: res.Code}
	if res.Code == 0 && c.Rafraichir != nil {
		if err := c.Rafraichir(ctx); err != nil {
			return sortie, fmt.Errorf("la commande a réussi, mais le tableau de bord n'a pas pu être mis à jour : %w", err)
		}
	}
	return sortie, nil
}
