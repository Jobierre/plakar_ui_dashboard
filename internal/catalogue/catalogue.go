// Package catalogue tient la liste des Stores et, pour chacun, le résultat de
// son dernier Relevé et son Seuil de retard propre. Il conserve cet état dans
// un fichier, qui survit à un redémarrage du service.
package catalogue

import (
	"errors"
	"fmt"
	"maps"
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
	// enCours donne, pour chaque Store dont un Relevé est commencé et pas
	// encore enregistré, le jeton de ce Relevé. Il n'est pas conservé : un
	// Relevé ne survit pas à un redémarrage.
	enCours map[string]Jeton
	// dernier est le dernier jeton attribué.
	dernier Jeton
}

// Jeton désigne un Relevé commencé par Begin. Seul le Relevé en cours d'un
// Store peut être enregistré ou abandonné : un Relevé lancé avant le retrait
// du Store ne compte plus, même si le Store revient entre-temps.
type Jeton uint64

// Entry est un Store, le résultat de son dernier Relevé et son Seuil.
type Entry struct {
	Name string
	// Releve est nil tant qu'aucun Relevé du Store n'est terminé.
	Releve *releve.Result
	// Seuil est le Seuil de retard propre au Store, nil s'il suit le Seuil
	// par défaut.
	Seuil *time.Duration
	// EnCours indique qu'un Relevé du Store est commencé et pas encore
	// terminé.
	EnCours bool
}

// ErrUnknownStore signale un Store absent du Catalogue.
var ErrUnknownStore = errors.New("Store inconnu")

// Open crée le Catalogue de ces Stores, dans cet ordre, et reprend dans le
// fichier d'état path leurs Seuils et leur dernier Relevé. Un fichier absent
// est un état vide ; un fichier illisible est une erreur, pour ne jamais
// écraser des Seuils qu'on n'a pas su lire.
func Open(path string, names []string) (*Catalogue, error) {
	c := &Catalogue{path: path, names: names, results: map[string]releve.Result{}, seuils: map[string]time.Duration{}, enCours: map[string]Jeton{}}
	if err := c.load(); err != nil {
		return nil, fmt.Errorf("fichier d'état %s : %w", path, err)
	}
	return c, nil
}

// Begin marque le début d'un Relevé du Store et renvoie son jeton. Il renvoie
// false si un Relevé de ce Store est déjà en cours : deux Relevés d'un même
// Store ne se chevauchent jamais. Le Relevé se termine par Record ou Abandon.
func (c *Catalogue) Begin(name string) (Jeton, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !slices.Contains(c.names, name) {
		return 0, false, ErrUnknownStore
	}
	if _, ok := c.enCours[name]; ok {
		return 0, false, nil
	}
	c.dernier++
	c.enCours[name] = c.dernier
	return c.dernier, true, nil
}

// Record termine le Relevé en cours d'un Store, garde son résultat et
// l'enregistre. Un Relevé en échec garde les dernières valeurs connues du
// Store, à côté de l'erreur. Le résultat d'un Relevé qui n'est plus le Relevé
// en cours du Store (le Store a été retiré entre-temps) est ignoré. L'erreur
// renvoyée est celle de l'enregistrement : le résultat reste affiché.
func (c *Catalogue) Record(name string, j Jeton, r releve.Result) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.enCours[name] != j {
		return nil
	}
	if prev, ok := c.results[name]; ok && r.Err != nil {
		r.Latest, r.Previous, r.Ecart = prev.Latest, prev.Previous, prev.Ecart
	}
	delete(c.enCours, name)
	c.results[name] = r
	return c.save()
}

// SetStores remplace la liste des Stores par celle-ci, lue dans la config
// plakar, dans cet ordre. Un Store retiré perd son Seuil et son dernier
// Relevé, qui disparaissent du fichier d'état ; un Store ajouté n'a encore
// aucun Relevé. La liste est changée même si l'enregistrement échoue : la
// config plakar fait foi, et le fichier d'état sera repris à l'enregistrement
// suivant.
func (c *Catalogue) SetStores(names []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.names = slices.Clone(names)
	removed := func(name string) bool { return !slices.Contains(names, name) }
	maps.DeleteFunc(c.results, func(name string, _ releve.Result) bool { return removed(name) })
	maps.DeleteFunc(c.seuils, func(name string, _ time.Duration) bool { return removed(name) })
	// Le Relevé en cours d'un Store retiré ne compte plus : s'il revient,
	// sous une autre config peut-être, son premier Relevé est un nouveau.
	maps.DeleteFunc(c.enCours, func(name string, _ Jeton) bool { return removed(name) })
	return c.save()
}

// Abandon termine le Relevé en cours d'un Store sans en garder de résultat.
func (c *Catalogue) Abandon(name string, j Jeton) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.enCours[name] == j {
		delete(c.enCours, name)
	}
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
		_, enCours := c.enCours[name]
		entries[i] = Entry{Name: name, EnCours: enCours}
		if r, ok := c.results[name]; ok {
			entries[i].Releve = &r
		}
		if d, ok := c.seuils[name]; ok {
			entries[i].Seuil = &d
		}
	}
	return entries
}
