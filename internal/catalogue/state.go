package catalogue

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"plakar-ui/internal/etat"
	"plakar-ui/internal/releve"
)

// stateFile est le contenu du fichier d'état.
type stateFile struct {
	Stores map[string]storeState `json:"stores"`
	// Groupes sont les Groupes, dans l'ordre, et SansGroupe les Stores rangés
	// nulle part, dans l'ordre. Un fichier qui n'en a pas (version
	// précédente) range tous ses Stores dans « Sans groupe ».
	Groupes    []savedGroupe `json:"groupes"`
	SansGroupe []string      `json:"sans_groupe"`
}

// savedGroupe est un Groupe enregistré : son nom et ses membres, dans l'ordre.
type savedGroupe struct {
	Nom    string   `json:"nom"`
	Stores []string `json:"stores"`
}

// storeState est ce que le service sait d'un Store, en dehors de la config
// plakar.
type storeState struct {
	// Seuil est le Seuil de retard propre au Store (« 8d »), vide s'il suit
	// le Seuil par défaut.
	Seuil  string       `json:"seuil,omitempty"`
	Releve *savedReleve `json:"releve,omitempty"`
	// Port est le port d'Exploration attribué au Store.
	Port int `json:"port,omitempty"`
}

// savedReleve est un releve.Result enregistrable : l'erreur y est un texte.
type savedReleve struct {
	At       time.Time        `json:"at"`
	Latest   *releve.Snapshot `json:"latest,omitempty"`
	Previous *releve.Snapshot `json:"previous,omitempty"`
	Ecart    *releve.Ecart    `json:"ecart,omitempty"`
	Error    string           `json:"error,omitempty"`
}

// load reprend l'état des Stores du Catalogue. L'état des Stores qui ne sont
// plus dans la config plakar est ignoré, et disparaît à l'enregistrement
// suivant.
func (c *Catalogue) load() error {
	raw, err := os.ReadFile(c.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var f stateFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	for _, g := range f.Groupes {
		c.groupes = append(c.groupes, groupe{nom: g.Nom, membres: slices.Clone(g.Stores)})
	}
	c.sansGroupe = slices.Clone(f.SansGroupe)
	for name, s := range f.Stores {
		if !slices.Contains(c.names, name) {
			continue
		}
		if s.Seuil != "" {
			d, err := etat.ParseSeuil(s.Seuil)
			if err != nil {
				return fmt.Errorf("Store %s : %w", name, err)
			}
			c.seuils[name] = d
		}
		if s.Port != 0 {
			c.ports[name] = s.Port
		}
		if r := s.Releve; r != nil {
			res := releve.Result{At: r.At, Latest: r.Latest, Previous: r.Previous, Ecart: r.Ecart}
			if r.Error != "" {
				res.Err = errors.New(r.Error)
			}
			c.results[name] = res
		}
	}
	return nil
}

// save enregistre l'état des Stores. L'écriture est atomique : le fichier est
// écrit à côté, puis renommé, si bien qu'un arrêt brutal laisse l'ancien
// fichier ou le nouveau, jamais un fichier tronqué. Il faut tenir c.mu.
func (c *Catalogue) save() error {
	f := stateFile{Stores: map[string]storeState{}, Groupes: []savedGroupe{}, SansGroupe: append([]string{}, c.sansGroupe...)} // [] et non null, même vide
	for _, g := range c.groupes {
		f.Groupes = append(f.Groupes, savedGroupe{Nom: g.nom, Stores: slices.Clone(g.membres)})
	}
	for _, name := range c.names {
		var s storeState
		if d, ok := c.seuils[name]; ok {
			s.Seuil = etat.FormatSeuil(d)
		}
		if r, ok := c.results[name]; ok {
			s.Releve = &savedReleve{At: r.At, Latest: r.Latest, Previous: r.Previous, Ecart: r.Ecart}
			if r.Err != nil {
				s.Releve.Error = r.Err.Error()
			}
		}
		s.Port = c.ports[name]
		if s != (storeState{}) {
			f.Stores[name] = s
		}
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(c.path, append(raw, '\n'))
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // sans effet une fois renommé
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return err
	}
	// Le renommage lui-même doit survivre à une coupure.
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
