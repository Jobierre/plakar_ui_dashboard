package catalogue

import (
	"errors"
	"fmt"
	"maps"
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
// Store retiré reste dans son Groupe comme Store disparu, mais quitte « Sans
// groupe » ; un nom rangé deux fois ne garde que sa première place, et un
// Store rangé nulle part va à la fin de « Sans groupe », par ordre de nom. Il
// dit si le rangement a changé. Il faut tenir c.mu.
func (c *Catalogue) reconcilier() bool {
	changed := false
	place := map[string]bool{}
	garder := func(membres []string, gardeDisparus bool) []string {
		out := membres[:0]
		for _, name := range membres {
			if place[name] || !gardeDisparus && !slices.Contains(c.names, name) {
				changed = true
				continue
			}
			place[name] = true
			out = append(out, name)
		}
		return out
	}
	for i := range c.groupes {
		c.groupes[i].membres = garder(c.groupes[i].membres, true)
	}
	c.sansGroupe = garder(c.sansGroupe, false)
	nouveaux := slices.DeleteFunc(slices.Clone(c.names), func(name string) bool { return place[name] })
	slices.Sort(nouveaux)
	if len(nouveaux) > 0 {
		c.sansGroupe = append(c.sansGroupe, nouveaux...)
		changed = true
	}
	return changed
}

// ranger applique un changement du rangement et l'enregistre. Un Store
// disparu qui quitte son Groupe est oublié, Seuil compris. Si le changement
// est refusé ou si l'enregistrement échoue, rien n'est changé : un rangement
// n'est jamais à moitié fait.
func (c *Catalogue) ranger(change func() error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	groupes, sans := c.cloneRangement()
	seuils := maps.Clone(c.seuils)
	annuler := func() { c.groupes, c.sansGroupe, c.seuils = groupes, sans, seuils }
	if err := change(); err != nil {
		annuler()
		return err
	}
	c.oublierSeuils()
	if err := c.save(); err != nil {
		annuler()
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
// non vide, différent du mot réservé et de celui de tout autre Groupe que
// celui à la place sauf (-1 : aucun). Il faut tenir c.mu.
func (c *Catalogue) nomGroupe(nom string, sauf int) (string, error) {
	propre := strings.TrimSpace(nom)
	switch i := c.groupe(propre); {
	case propre == "":
		return "", &NomRefuse{Nom: nom, Raison: "Un Groupe doit avoir un nom."}
	case propre == SansGroupe:
		return "", &NomRefuse{Nom: nom, Raison: fmt.Sprintf("« %s » est réservé à « Sans groupe ».", SansGroupe)}
	case i >= 0 && i != sauf:
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
		propre, err := c.nomGroupe(nom, -1)
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
		cible, err := c.cible(store, nom)
		if err != nil || slices.Contains(*cible, store) {
			return err
		}
		c.retirer(store)
		*cible = append(*cible, store)
		return nil
	})
}

// Placer range le Store dans ce Groupe, ou dans « Sans groupe » (SansGroupe),
// juste avant le Store avant, ou à la fin si avant est vide, et l'enregistre.
// Le Store peut être déjà dans ce Groupe : il y change de place. Placé avant
// lui-même, il garde sa place. Un Store avant qui n'est pas dans ce Groupe est
// refusé par ErrPositionInconnue.
func (c *Catalogue) Placer(store, nom, avant string) error {
	return c.ranger(func() error {
		cible, err := c.cible(store, nom)
		if err != nil {
			return err
		}
		if avant != "" && !slices.Contains(*cible, avant) {
			return ErrPositionInconnue
		}
		if avant == store {
			return nil
		}
		c.retirer(store)
		i := len(*cible)
		if avant != "" {
			i = slices.Index(*cible, avant)
		}
		*cible = slices.Insert(*cible, i, store)
		return nil
	})
}

// ErrPositionInconnue signale un Store avant lequel placer qui n'est pas dans
// le Groupe visé, ou un Groupe avant lequel placer qui n'existe pas.
var ErrPositionInconnue = errors.New("position inconnue")

// cible renvoie la liste où ranger ce Store du Catalogue : les membres du
// Groupe qui porte ce nom, ou « Sans groupe » (SansGroupe). Il faut tenir c.mu.
func (c *Catalogue) cible(store, nom string) (*[]string, error) {
	if !slices.Contains(c.names, store) {
		return nil, ErrUnknownStore
	}
	if nom == SansGroupe {
		return &c.sansGroupe, nil
	}
	i, err := c.groupeVise(nom)
	if err != nil {
		return nil, err
	}
	return &c.groupes[i].membres, nil
}

// retirer ôte le Store de son Groupe, ou de « Sans groupe ». Il faut tenir
// c.mu.
func (c *Catalogue) retirer(store string) {
	if l := c.rangementDe(store); l != nil {
		*l = slices.DeleteFunc(*l, func(m string) bool { return m == store })
	}
}

// rangementDe renvoie la liste où le Store est rangé : les membres de son Groupe,
// ou « Sans groupe ». Il faut tenir c.mu.
func (c *Catalogue) rangementDe(store string) *[]string {
	for i := range c.groupes {
		if slices.Contains(c.groupes[i].membres, store) {
			return &c.groupes[i].membres
		}
	}
	if slices.Contains(c.sansGroupe, store) {
		return &c.sansGroupe
	}
	return nil
}

// Sens est le sens d'un déplacement d'une place : Haut (↑) ou Bas (↓).
type Sens int

const (
	Haut Sens = -1
	Bas  Sens = 1
)

// DeplacerStore déplace le Store d'une place dans son Groupe, ou dans « Sans
// groupe », vers le haut (Haut) ou vers le bas (Bas), et l'enregistre. Le
// premier Store ne monte pas plus haut, ni le dernier plus bas.
func (c *Catalogue) DeplacerStore(store string, sens Sens) error {
	return c.ranger(func() error {
		l := c.rangementDe(store)
		if l == nil {
			return ErrUnknownStore
		}
		echanger(*l, slices.Index(*l, store), sens)
		return nil
	})
}

// echanger échange l'élément i avec son voisin dans ce sens, s'il en a un.
func echanger[T any](s []T, i int, sens Sens) {
	if j := i + int(sens); j >= 0 && j < len(s) {
		s[i], s[j] = s[j], s[i]
	}
}

// ErrSansGroupeFixe signale une action qui viserait « Sans groupe » lui-même :
// il ne se renomme, ne se supprime ni ne se déplace.
var ErrSansGroupeFixe = errors.New("« Sans groupe » ne se renomme, ne se supprime ni ne se déplace.")

// groupeVise renvoie la place du Groupe sur lequel porte une action : un
// Groupe du Catalogue, jamais « Sans groupe ». Il faut tenir c.mu.
func (c *Catalogue) groupeVise(nom string) (int, error) {
	if nom == SansGroupe {
		return 0, ErrSansGroupeFixe
	}
	i := c.groupe(nom)
	if i < 0 {
		return 0, ErrUnknownGroupe
	}
	return i, nil
}

// DeplacerGroupe déplace le Groupe d'une place parmi les Groupes, vers le haut
// (Haut) ou vers le bas (Bas), et l'enregistre. « Sans groupe » reste en
// dernier.
func (c *Catalogue) DeplacerGroupe(nom string, sens Sens) error {
	return c.ranger(func() error {
		i, err := c.groupeVise(nom)
		if err != nil {
			return err
		}
		echanger(c.groupes, i, sens)
		return nil
	})
}

// PlacerGroupe déplace le Groupe juste avant le Groupe avant, ou en dernier
// (juste avant « Sans groupe ») si avant est vide ou SansGroupe, et
// l'enregistre. Placé avant lui-même, il garde sa place. Un Groupe avant qui
// n'existe pas est refusé par ErrPositionInconnue.
func (c *Catalogue) PlacerGroupe(nom, avant string) error {
	return c.ranger(func() error {
		i, err := c.groupeVise(nom)
		if err != nil {
			return err
		}
		enDernier := avant == "" || avant == SansGroupe
		if !enDernier && c.groupe(avant) < 0 {
			return ErrPositionInconnue
		}
		if avant == nom {
			return nil
		}
		g := c.groupes[i]
		c.groupes = slices.Delete(c.groupes, i, i+1)
		j := len(c.groupes)
		if !enDernier {
			j = c.groupe(avant)
		}
		c.groupes = slices.Insert(c.groupes, j, g)
		return nil
	})
}

// RenommerGroupe donne au Groupe ce nouveau nom, aux mêmes règles qu'à la
// création, et l'enregistre. Le Groupe garde sa place et ses Stores.
func (c *Catalogue) RenommerGroupe(nom, nouveau string) error {
	return c.ranger(func() error {
		i, err := c.groupeVise(nom)
		if err != nil {
			return err
		}
		propre, err := c.nomGroupe(nouveau, i)
		if err != nil {
			return err
		}
		c.groupes[i].nom = propre
		return nil
	})
}

// SupprimerGroupe supprime le Groupe et envoie ses Stores à la fin de « Sans
// groupe », dans leur ordre, et l'enregistre. Ses Stores disparus sont
// oubliés, Seuil compris.
func (c *Catalogue) SupprimerGroupe(nom string) error {
	return c.ranger(func() error {
		i, err := c.groupeVise(nom)
		if err != nil {
			return err
		}
		c.sansGroupe = append(c.sansGroupe, slices.DeleteFunc(slices.Clone(c.groupes[i].membres), c.disparu)...)
		c.groupes = slices.Delete(c.groupes, i, i+1)
		return nil
	})
}

// ErrPasDisparu signale un Store de la config visé comme Store disparu : il se
// range, il ne se retire pas.
var ErrPasDisparu = errors.New("pas un Store disparu")

// RetirerDisparu retire le Store disparu de son Groupe, l'oublie, Seuil
// compris, et l'enregistre. Un Store de la config est refusé par
// ErrPasDisparu, un nom inconnu par ErrUnknownStore.
func (c *Catalogue) RetirerDisparu(store string) error {
	return c.ranger(func() error {
		switch {
		case slices.Contains(c.names, store):
			return ErrPasDisparu
		case !c.disparu(store):
			return ErrUnknownStore
		}
		c.retirer(store)
		return nil
	})
}

// Membres renvoie les Stores du Groupe qui porte ce nom, ou de « Sans groupe »
// (SansGroupe), dans l'ordre, sans ses Stores disparus. Un Groupe inconnu est
// refusé par ErrUnknownGroupe.
func (c *Catalogue) Membres(nom string) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if nom == SansGroupe {
		return slices.Clone(c.sansGroupe), nil
	}
	i := c.groupe(nom)
	if i < 0 {
		return nil, ErrUnknownGroupe
	}
	return slices.DeleteFunc(slices.Clone(c.groupes[i].membres), c.disparu), nil
}
