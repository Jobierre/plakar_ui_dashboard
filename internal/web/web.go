// Package web sert le tableau de bord.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"plakar-ui/internal/catalogue"
)

//go:embed templates/*.html
var templates embed.FS

var page = template.Must(template.ParseFS(templates, "templates/dashboard.html"))

// Handler sert le tableau de bord des Stores du Catalogue. now donne l'heure
// courante et loc le fuseau de l'heure locale affichée.
func Handler(cat *catalogue.Catalogue, now func() time.Time, loc *time.Location, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		rows := make([]row, 0)
		for _, e := range cat.Entries() {
			rows = append(rows, newRow(e, now(), loc))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, rows); err != nil {
			log.Error("rendu du tableau de bord", "err", err)
		}
	})
	return mux
}

// row est une ligne du tableau de bord, prête à afficher.
type row struct {
	Name    string
	Pending bool
	Latest  *snapshot
	// Message remplace le Dernier snapshot quand il n'y en a pas.
	Message string
	// Error est le message d'erreur du dernier Relevé, s'il a échoué.
	Error string
}

// snapshot est le Dernier snapshot d'une ligne, mis en forme.
type snapshot struct {
	Date, Age, ID, Size, Duration, Path, Tags string
}

func newRow(e catalogue.Entry, now time.Time, loc *time.Location) row {
	r := row{Name: e.Name}
	switch {
	case e.Releve == nil:
		r.Pending, r.Message = true, "Relevé en attente"
		return r
	case e.Releve.Err != nil:
		r.Message, r.Error = "Relevé en échec", e.Releve.Err.Error()
		return r
	case e.Releve.Latest == nil:
		r.Message = "Aucun snapshot"
		return r
	}
	s := e.Releve.Latest
	r.Latest = &snapshot{
		Date:     s.Date.In(loc).Format("2006-01-02 15:04"),
		Age:      age(now.Sub(s.Date)),
		ID:       s.ID,
		Size:     s.Size.Text,
		Duration: s.Duration.String(),
		Path:     s.Path,
		Tags:     strings.Join(s.Tags, ", "),
	}
	return r
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
