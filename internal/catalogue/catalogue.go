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
	// plage est la plage des ports d'Exploration, ports le port attribué à
	// chaque Store qui en a un.
	plage Plage
	ports map[string]int
	// enCours donne, pour chaque Store dont un Relevé est commencé et pas
	// encore enregistré, le jeton de ce Relevé. Il n'est pas conservé : un
	// Relevé ne survit pas à un redémarrage.
	enCours map[string]Jeton
	// dernier est le dernier jeton attribué.
	dernier Jeton
	// groupes sont les Groupes, dans l'ordre, et sansGroupe les Stores rangés
	// nulle part, dans l'ordre. Chaque Store de la config est dans l'un des
	// deux, une seule fois. Un membre de Groupe absent de la config est un
	// Store disparu ; « Sans groupe » n'en a jamais.
	groupes    []groupe
	sansGroupe []string
}

// groupe est un Groupe : son nom et ses membres, dans l'ordre.
type groupe struct {
	nom     string
	membres []string
}

// Groupe est un Groupe et ses Stores, dans l'ordre.
type Groupe struct {
	Nom    string
	Stores []Entry
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
	// Port est le port d'Exploration du Store, 0 si la plage est épuisée.
	Port int
	// Disparu marque un Store disparu : retiré de la config plakar alors qu'il
	// était rangé dans un Groupe. Il n'a ni Relevé, ni port ; seul son Seuil
	// est gardé.
	Disparu bool
}

// ErrUnknownStore signale un Store absent du Catalogue.
var ErrUnknownStore = errors.New("Store inconnu")

// Open crée le Catalogue de ces Stores, dans cet ordre, et reprend dans le
// fichier d'état path leurs Seuils, leur dernier Relevé et leur port
// d'Exploration. Un fichier absent est un état vide ; un fichier illisible est
// une erreur, pour ne jamais écraser des Seuils qu'on n'a pas su lire. Chaque
// Store sans port en reçoit un de la plage, et l'état est enregistré.
func Open(path string, names []string, plage Plage) (*Catalogue, error) {
	c := &Catalogue{path: path, names: names, results: map[string]releve.Result{}, seuils: map[string]time.Duration{}, enCours: map[string]Jeton{}, plage: plage, ports: map[string]int{}}
	if err := c.load(); err != nil {
		return nil, fmt.Errorf("fichier d'état %s : %w", path, err)
	}
	// Les deux doivent être faits : pas de court-circuit.
	ranges, ports := c.reconcilier(), c.assignPorts()
	if ranges || ports {
		if err := c.save(); err != nil {
			return nil, fmt.Errorf("fichier d'état %s : %w", path, err)
		}
	}
	return c, nil
}

// assignPorts donne à chaque Store sans port, dans l'ordre des Stores, le
// premier port libre de la plage. Un port hors de la plage (la plage a changé)
// ou déjà pris par un Store précédent est rendu. Il dit si un port a changé.
// Il faut tenir c.mu.
func (c *Catalogue) assignPorts() bool {
	changed := false
	used := map[int]bool{}
	for _, name := range c.names {
		port, ok := c.ports[name]
		if !ok {
			continue
		}
		if !c.plage.Contains(port) || used[port] {
			delete(c.ports, name)
			changed = true
			continue
		}
		used[port] = true
	}
	free := slices.DeleteFunc(c.plage.Ports(), func(p int) bool { return used[p] })
	for _, name := range c.names {
		if _, ok := c.ports[name]; ok || len(free) == 0 {
			continue
		}
		c.ports[name], free = free[0], free[1:]
		changed = true
	}
	return changed
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
// plakar, dans cet ordre. Un Store retiré perd son dernier Relevé et son port,
// qui disparaissent du fichier d'état ; rangé dans un Groupe, il y reste comme
// Store disparu et garde son Seuil, sinon il perd aussi son Seuil. Un Store
// ajouté n'a encore aucun Relevé, et reçoit un port ; s'il était un Store
// disparu, il reprend sa place et son Seuil. La liste est changée même si
// l'enregistrement échoue : la config plakar fait foi, et le fichier d'état
// sera repris à l'enregistrement suivant.
func (c *Catalogue) SetStores(names []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.names = slices.Clone(names)
	c.reconcilier()
	removed := func(name string) bool { return !slices.Contains(names, name) }
	maps.DeleteFunc(c.results, func(name string, _ releve.Result) bool { return removed(name) })
	c.oublierSeuils()
	// Le port d'un Store retiré est libéré, et peut servir à un Store ajouté.
	maps.DeleteFunc(c.ports, func(name string, _ int) bool { return removed(name) })
	c.assignPorts()
	// Le Relevé en cours d'un Store retiré ne compte plus : s'il revient,
	// sous une autre config peut-être, son premier Relevé est un nouveau.
	maps.DeleteFunc(c.enCours, func(name string, _ Jeton) bool { return removed(name) })
	return c.save()
}

// oublierSeuils oublie le Seuil de chaque nom qui n'est ni un Store, ni un
// Store disparu. Il faut tenir c.mu.
func (c *Catalogue) oublierSeuils() {
	maps.DeleteFunc(c.seuils, func(name string, _ time.Duration) bool {
		return !slices.Contains(c.names, name) && !c.disparu(name)
	})
}

// disparu dit si ce nom est celui d'un Store disparu : membre d'un Groupe,
// absent de la config. Il faut tenir c.mu.
func (c *Catalogue) disparu(name string) bool {
	if slices.Contains(c.names, name) {
		return false
	}
	return slices.ContainsFunc(c.groupes, func(g groupe) bool { return slices.Contains(g.membres, name) })
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

// Entry renvoie le Store qui porte ce nom, et son dernier Relevé.
func (c *Catalogue) Entry(name string) (Entry, bool) {
	for _, e := range c.Entries() {
		if e.Name == name {
			return e, true
		}
	}
	return Entry{}, false
}

// Entries renvoie les Stores et leur dernier Relevé, dans l'ordre.
func (c *Catalogue) Entries() []Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries := make([]Entry, len(c.names))
	for i, name := range c.names {
		entries[i] = c.entry(name)
	}
	return entries
}

// entry renvoie le Store, ou le Store disparu, qui porte ce nom. Il faut
// tenir c.mu.
func (c *Catalogue) entry(name string) Entry {
	e := Entry{Name: name}
	if d, ok := c.seuils[name]; ok {
		e.Seuil = &d
	}
	// Un Store disparu n'a que son Seuil.
	if c.disparu(name) {
		e.Disparu = true
		return e
	}
	_, e.EnCours = c.enCours[name]
	e.Port = c.ports[name]
	if r, ok := c.results[name]; ok {
		e.Releve = &r
	}
	return e
}

// Groupes renvoie les Groupes et leurs Stores, Stores disparus compris, dans
// l'ordre, puis les Stores « Sans groupe », dans l'ordre.
func (c *Catalogue) Groupes() (groupes []Groupe, sansGroupe []Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries := func(names []string) []Entry {
		out := make([]Entry, len(names))
		for i, name := range names {
			out[i] = c.entry(name)
		}
		return out
	}
	groupes = make([]Groupe, len(c.groupes))
	for i, g := range c.groupes {
		groupes[i] = Groupe{Nom: g.nom, Stores: entries(g.membres)}
	}
	return groupes, entries(c.sansGroupe)
}
