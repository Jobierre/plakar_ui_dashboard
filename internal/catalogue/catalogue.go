// Package catalogue tient la liste des Stores et, pour chacun, le résultat de
// son dernier Relevé.
package catalogue

import (
	"sync"

	"plakar-ui/internal/releve"
)

// Catalogue est la liste des Stores, sûre pour un accès concurrent.
type Catalogue struct {
	mu      sync.Mutex
	names   []string
	results map[string]releve.Result
}

// Entry est un Store et le résultat de son dernier Relevé.
type Entry struct {
	Name string
	// Releve est nil tant qu'aucun Relevé du Store n'est terminé.
	Releve *releve.Result
}

// New crée un Catalogue de ces Stores, dans cet ordre.
func New(names []string) *Catalogue {
	return &Catalogue{names: names, results: map[string]releve.Result{}}
}

// Record garde le résultat du dernier Relevé d'un Store.
func (c *Catalogue) Record(name string, r releve.Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[name] = r
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
	}
	return entries
}
