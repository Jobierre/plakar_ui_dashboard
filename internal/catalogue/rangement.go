package catalogue

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// SansGroupe est le mot réservé qui désigne « Sans groupe » : aucun Groupe ne
// peut porter ce nom.
const SansGroupe = "_sans"

// ErrUnknownGroupe signale un Groupe absent du Catalogue.
var ErrUnknownGroupe = errors.New("Groupe inconnu")

// NomRefuse est un nom de Groupe refusé, et la raison du refus.
type NomRefuse struct {
	Nom, Raison string
}

func (e *NomRefuse) Error() string { return e.Raison }

// reconcilier accorde le rangement à la liste des Stores de la config : un
// Store retiré quitte son Groupe ou « Sans groupe », un Store rangé deux fois
// ne garde que sa première place, et un Store rangé nulle part va à la fin de
// « Sans groupe », par ordre de nom. Il dit si le rangement a changé. Il faut
// tenir c.mu.
func (c *Catalogue) reconcilier() bool {
	changed := false
	place := map[string]bool{}
	garder := func(membres []string) []string {
		out := membres[:0]
		for _, name := range membres {
			if place[name] || !slices.Contains(c.names, name) {
				changed = true
				continue
			}
			place[name] = true
			out = append(out, name)
		}
		return out
	}
	for i := range c.groupes {
		c.groupes[i].membres = garder(c.groupes[i].membres)
	}
	c.sansGroupe = garder(c.sansGroupe)
	nouveaux := slices.DeleteFunc(slices.Clone(c.names), func(name string) bool { return place[name] })
	slices.Sort(nouveaux)
	if len(nouveaux) > 0 {
		c.sansGroupe = append(c.sansGroupe, nouveaux...)
		changed = true
	}
	return changed
}

// ranger applique un changement du rangement et l'enregistre. Si le
// changement est refusé ou si l'enregistrement échoue, le rangement n'est pas
// changé : un rangement n'est jamais à moitié fait.
func (c *Catalogue) ranger(change func() error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	groupes, sans := c.cloneRangement()
	if err := change(); err != nil {
		c.groupes, c.sansGroupe = groupes, sans
		return err
	}
	if err := c.save(); err != nil {
		c.groupes, c.sansGroupe = groupes, sans
		return fmt.Errorf("rangement non enregistré : %w", err)
	}
	return nil
}

// cloneRangement copie le rangement, membres compris. Il faut tenir c.mu.
func (c *Catalogue) cloneRangement() ([]groupe, []string) {
	groupes := make([]groupe, len(c.groupes))
	for i, g := range c.groupes {
		groupes[i] = groupe{nom: g.nom, membres: slices.Clone(g.membres)}
	}
	return groupes, slices.Clone(c.sansGroupe)
}

// nomGroupe nettoie un nom de Groupe des espaces de bord et vérifie qu'il est
// non vide, différent du mot réservé et d'aucun autre Groupe. Il faut tenir
// c.mu.
func (c *Catalogue) nomGroupe(nom string) (string, error) {
	propre := strings.TrimSpace(nom)
	switch {
	case propre == "":
		return "", &NomRefuse{Nom: nom, Raison: "Un Groupe doit avoir un nom."}
	case propre == SansGroupe:
		return "", &NomRefuse{Nom: nom, Raison: fmt.Sprintf("« %s » est réservé à « Sans groupe ».", SansGroupe)}
	case c.groupe(propre) >= 0:
		return "", &NomRefuse{Nom: nom, Raison: fmt.Sprintf("Le Groupe « %s » existe déjà.", propre)}
	}
	return propre, nil
}

// groupe renvoie la place du Groupe qui porte ce nom, -1 s'il n'existe pas.
// Il faut tenir c.mu.
func (c *Catalogue) groupe(nom string) int {
	return slices.IndexFunc(c.groupes, func(g groupe) bool { return g.nom == nom })
}

// CreerGroupe crée un Groupe vide, après les autres, et l'enregistre. Le nom
// est nettoyé des espaces de bord ; un nom vide, réservé ou déjà pris est
// refusé par une *NomRefuse.
func (c *Catalogue) CreerGroupe(nom string) error {
	return c.ranger(func() error {
		propre, err := c.nomGroupe(nom)
		if err != nil {
			return err
		}
		c.groupes = append(c.groupes, groupe{nom: propre})
		return nil
	})
}

// Ranger range le Store dans ce Groupe, ou dans « Sans groupe » (SansGroupe),
// à la fin, et l'enregistre. Un Store déjà dans ce Groupe garde sa place.
func (c *Catalogue) Ranger(store, nom string) error {
	return c.ranger(func() error {
		if !slices.Contains(c.names, store) {
			return ErrUnknownStore
		}
		cible := &c.sansGroupe
		if nom != SansGroupe {
			i := c.groupe(nom)
			if i < 0 {
				return ErrUnknownGroupe
			}
			cible = &c.groupes[i].membres
		}
		if slices.Contains(*cible, store) {
			return nil
		}
		sans := func(membres []string) []string {
			return slices.DeleteFunc(membres, func(m string) bool { return m == store })
		}
		for i := range c.groupes {
			c.groupes[i].membres = sans(c.groupes[i].membres)
		}
		c.sansGroupe = sans(c.sansGroupe)
		*cible = append(*cible, store)
		return nil
	})
}
