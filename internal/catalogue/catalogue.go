// Package catalogue tient la liste des Stores et, pour chacun, le résultat de
// son dernier Relevé et son Seuil de retard propre. Il conserve cet état dans
// un fichier, qui survit à un redémarrage du service.
package catalogue

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"plakar-ui/internal/releve"
)

// Catalogue est la liste des Stores, sûre pour un accès concurrent.
type Catalogue struct {
	mu      sync.Mutex
	path    string // fichier d'état
	names   []string
	results map[string]releve.Result
	seuils  map[string]time.Duration
}

// Entry est un Store, le résultat de son dernier Relevé et son Seuil.
type Entry struct {
	Name string
	// Releve est nil tant qu'aucun Relevé du Store n'est terminé.
	Releve *releve.Result
	// Seuil est le Seuil de retard propre au Store, nil s'il suit le Seuil
	// par défaut.
	Seuil *time.Duration
}

// ErrUnknownStore signale un Store absent du Catalogue.
var ErrUnknownStore = errors.New("Store inconnu")

// Open crée le Catalogue de ces Stores, dans cet ordre, et reprend dans le
// fichier d'état path leurs Seuils et leur dernier Relevé. Un fichier absent
// est un état vide ; un fichier illisible est une erreur, pour ne jamais
// écraser des Seuils qu'on n'a pas su lire.
func Open(path string, names []string) (*Catalogue, error) {
	c := &Catalogue{path: path, names: names, results: map[string]releve.Result{}, seuils: map[string]time.Duration{}}
	if err := c.load(); err != nil {
		return nil, fmt.Errorf("fichier d'état %s : %w", path, err)
	}
	return c, nil
}

// Record garde le résultat du dernier Relevé d'un Store et l'enregistre. Un
// Relevé en échec garde les dernières valeurs connues du Store, à côté de
// l'erreur. L'erreur renvoyée est celle de l'enregistrement : le résultat
// reste affiché.
func (c *Catalogue) Record(name string, r releve.Result) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if prev, ok := c.results[name]; ok && r.Err != nil {
		r.Latest, r.Previous, r.Ecart = prev.Latest, prev.Previous, prev.Ecart
	}
	c.results[name] = r
	return c.save()
}

// SetSeuil donne au Store un Seuil de retard propre, ou lui rend le Seuil par
// défaut si seuil est nil, et l'enregistre. Si l'enregistrement échoue, le
// Seuil n'est pas changé.
func (c *Catalogue) SetSeuil(name string, seuil *time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !slices.Contains(c.names, name) {
		return ErrUnknownStore
	}
	old, had := c.seuils[name]
	if seuil == nil {
		delete(c.seuils, name)
	} else {
		c.seuils[name] = *seuil
	}
	if err := c.save(); err != nil {
		if had {
			c.seuils[name] = old
		} else {
			delete(c.seuils, name)
		}
		return fmt.Errorf("Seuil non enregistré : %w", err)
	}
	return nil
}

// Entries renvoie les Stores et leur dernier Relevé, dans l'ordre.
func (c *Catalogue) Entries() []Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries := make([]Entry, len(c.names))
	for i, name := range c.names {
		entries[i] = Entry{Name: name}
		if r, ok := c.results[name]; ok {
			entries[i].Releve = &r
		}
		if d, ok := c.seuils[name]; ok {
			entries[i].Seuil = &d
		}
	}
	return entries
}
