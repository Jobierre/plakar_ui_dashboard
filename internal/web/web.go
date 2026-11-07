// Package web sert le tableau de bord.
package web

import (
	"cmp"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"plakar-ui/internal/catalogue"
	"plakar-ui/internal/etat"
	"plakar-ui/internal/releve"
)

//go:embed templates/*.html
var templates embed.FS

var page = template.Must(template.ParseFS(templates, "templates/dashboard.html"))

// Dashboard est le tableau de bord et ce dont il a besoin.
type Dashboard struct {
	Catalogue *catalogue.Catalogue
	// DefaultSeuil est le Seuil de retard des Stores qui n'en ont pas de
	// propre.
	DefaultSeuil time.Duration
	// Trigger lance un Relevé immédiat d'un Store (bouton ↻), sans en
	// attendre la fin.
	Trigger func(store string) error
	// Now donne l'heure courante, Location le fuseau de l'heure affichée.
	Now      func() time.Time
	Location *time.Location
	Log      *slog.Logger
}

// Handler sert le tableau de bord des Stores du Catalogue, le changement de
// Seuil de retard d'un Store et son Relevé à la demande.
func Handler(c Dashboard) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		c.render(w, http.StatusOK, nil)
	})
	mux.HandleFunc("POST /stores/{store}/seuil", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("store")
		// Une valeur vide rend au Store le Seuil par défaut.
		var seuil *time.Duration
		if text := r.FormValue("seuil"); strings.TrimSpace(text) != "" {
			d, err := etat.ParseSeuil(text)
			if err != nil {
				c.render(w, http.StatusBadRequest, &seuilRefused{store: name, text: text, err: err.Error()})
				return
			}
			seuil = &d
		}
		if err := c.Catalogue.SetSeuil(name, seuil); err != nil {
			if errors.Is(err, catalogue.ErrUnknownStore) {
				http.NotFound(w, r)
				return
			}
			c.render(w, http.StatusInternalServerError, &seuilRefused{store: name, text: r.FormValue("seuil"), err: err.Error()})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /stores/{store}/releve", func(w http.ResponseWriter, r *http.Request) {
		if err := c.Trigger(r.PathValue("store")); err != nil {
			if errors.Is(err, catalogue.ErrUnknownStore) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// La page montre alors le Relevé en cours.
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	// Les formulaires ne sont acceptés que depuis le tableau de bord lui-même,
	// et non depuis une page tierce ouverte dans le même navigateur.
	return http.NewCrossOriginProtection().Handler(mux)
}

// seuilRefused est un Seuil de retard refusé, à signaler sur la ligne de son
// Store.
type seuilRefused struct {
	store, text, err string
}

// render écrit le tableau de bord, avec le Seuil refusé éventuel.
func (c Dashboard) render(w http.ResponseWriter, status int, refused *seuilRefused) {
	now := c.Now()
	rows := make([]row, 0)
	for _, e := range c.Catalogue.Entries() {
		r := c.newRow(e, now)
		if refused != nil && refused.store == e.Name {
			r.SeuilInput, r.SeuilError = refused.text, refused.err
		}
		rows = append(rows, r)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := page.Execute(w, rows); err != nil {
		c.Log.Error("rendu du tableau de bord", "err", err)
	}
}

// row est une ligne du tableau de bord, prête à afficher.
type row struct {
	Name    string
	Pending bool
	// EnCours indique qu'un Relevé du Store est en cours ; ReleveAt est
	// l'heure du dernier Relevé terminé, vide s'il n'y en a pas.
	EnCours  bool
	ReleveAt string
	// Etat est vide tant qu'aucun Relevé n'est terminé.
	Etat  string
	Seuil string
	// SeuilInput est la valeur du champ de Seuil, vide pour le Seuil par
	// défaut ; SeuilError explique pourquoi une valeur saisie a été refusée.
	SeuilInput, SeuilError string
	Latest                 *snapshot
	// Message remplace le Dernier snapshot quand il n'y en a pas.
	Message string
	// Error est le message d'erreur du dernier Relevé, s'il a échoué.
	Error string
}

// dateLayout est le format des dates affichées, en heure locale.
const dateLayout = "2006-01-02 15:04"

// snapshot est le Dernier snapshot d'une ligne, mis en forme, avec l'Écart
// (vide sans Snapshot précédent).
type snapshot struct {
	Date, Age, ID, Size, Duration, Path, Tags string
	SizeEcart, DurationEcart                  string
	// Previous décrit le Snapshot précédent, auquel l'Écart compare.
	Previous string
}

func (c Dashboard) newRow(e catalogue.Entry, now time.Time) row {
	seuil := c.DefaultSeuil
	r := row{Name: e.Name, EnCours: e.EnCours, Seuil: etat.FormatSeuil(seuil) + " (par défaut)"}
	if e.Seuil != nil {
		seuil = *e.Seuil
		r.Seuil = etat.FormatSeuil(seuil)
		r.SeuilInput = r.Seuil
	}
	if e.Releve != nil {
		r.Etat = etat.Of(e.Releve.Latest, seuil, now, e.Releve.Err).String()
		r.ReleveAt = e.Releve.At.In(c.Location).Format(dateLayout)
	}
	switch {
	case e.Releve == nil:
		r.Pending, r.Message = true, "Aucun Relevé terminé"
		return r
	}
	// Un Store Injoignable garde ses dernières valeurs connues, à côté de
	// l'erreur.
	if e.Releve.Err != nil {
		r.Error = e.Releve.Err.Error()
	}
	switch {
	case e.Releve.Latest == nil && e.Releve.Err != nil:
		r.Message = "Relevé en échec"
		return r
	case e.Releve.Latest == nil:
		r.Message = "Aucun snapshot"
		return r
	}
	s := e.Releve.Latest
	r.Latest = &snapshot{
		Date:     s.Date.In(c.Location).Format(dateLayout),
		Age:      age(now.Sub(s.Date)),
		ID:       s.ID,
		Size:     s.Size.Text,
		Duration: s.Duration.String(),
		Path:     s.Path,
		Tags:     strings.Join(s.Tags, ", "),
	}
	// Le Relevé ne produit un Écart qu'avec un Snapshot précédent.
	if ec, p := e.Releve.Ecart, e.Releve.Previous; ec != nil {
		r.Latest.Previous = fmt.Sprintf("par rapport au Snapshot précédent %s (%s, %s)", p.ID, p.Size.Text, p.Duration)
		r.Latest.SizeEcart = sizeEcart(*ec)
		r.Latest.DurationEcart = signed(cmp.Compare(ec.Duration, 0), ec.Duration.Abs().String())
	}
	return r
}

// sizeEcart écrit l'Écart de taille en pourcentage arrondi : « −97 % ».
func sizeEcart(e releve.Ecart) string {
	if e.SizeFromZero {
		return "nouveau"
	}
	pct := math.Round(e.Size * 100)
	return signed(cmp.Compare(pct, 0), fmt.Sprintf("%.0f\u00a0%%", math.Abs(pct)))
}

// signed préfixe une valeur absolue du signe donné par sign (−1, 0 ou 1) :
// « + », « − », ou rien si elle est nulle.
func signed(sign int, abs string) string {
	switch sign {
	case -1:
		return "−" + abs
	case 1:
		return "+" + abs
	default:
		return abs
	}
}

// age écrit un âge relatif : « il y a 6 h ».
func age(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "à l'instant"
	case d < time.Hour:
		return fmt.Sprintf("il y a %d min", int(d/time.Minute))
	case d < 48*time.Hour:
		return fmt.Sprintf("il y a %d h", int(d/time.Hour))
	default:
		return fmt.Sprintf("il y a %d j", int(d/(24*time.Hour)))
	}
}
