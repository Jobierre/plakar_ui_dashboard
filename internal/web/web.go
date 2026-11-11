// Package web sert le tableau de bord.
package web

import (
	"cmp"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"plakar-ui/internal/acces"
	"plakar-ui/internal/catalogue"
	"plakar-ui/internal/commande"
	"plakar-ui/internal/etat"
	"plakar-ui/internal/releve"
)

//go:embed templates/*.html
var templates embed.FS

// static sont les fichiers servis tels quels au navigateur, intégrés au
// binaire : l'interface ne dépend d'aucun CDN.
//
//go:embed static
var static embed.FS

var page = template.Must(template.New("dashboard.html").Funcs(template.FuncMap{
	// ecartDe prépare le bloc « ecart » : un Écart, le champ qui le nomme et
	// la description du Snapshot précédent, en infobulle.
	"ecartDe": func(field string, e *ecart, previous string) any {
		return struct {
			Field, Previous string
			Ecart           *ecart
		}{field, previous, e}
	},
}).ParseFS(templates, "templates/dashboard.html"))

// Dashboard est le tableau de bord et ce dont il a besoin.
type Dashboard struct {
	Catalogue *catalogue.Catalogue
	// DefaultSeuil est le Seuil de retard des Stores qui n'en ont pas de
	// propre.
	DefaultSeuil time.Duration
	// Trigger lance un Relevé immédiat d'un Store (bouton ↻), sans en
	// attendre la fin.
	Trigger func(store string) error
	// Explorations lance, donne et arrête les Explorations des Stores.
	Explorations Explorations
	// Commande exécute une Commande de Store saisie dans son bloc.
	Commande func(ctx context.Context, line string) (*commande.Sortie, error)
	// Now donne l'heure courante, Location le fuseau de l'heure affichée.
	Now      func() time.Time
	Location *time.Location
	Log      *slog.Logger
}

// Explorations sont les Explorations des Stores.
type Explorations interface {
	// Lancer lance l'Exploration du Store sur son port public, ou retrouve
	// celle en cours, et renvoie son lien.
	Lancer(store string, port int) (string, error)
	// Lien est le lien de l'Exploration en cours du Store, vide sans
	// Exploration.
	Lien(store string) string
	// Arreter arrête l'Exploration du Store, s'il y en a une.
	Arreter(store string)
}

// Handler sert le tableau de bord des Stores du Catalogue, le changement de
// Seuil de retard d'un Store et son Relevé à la demande.
func Handler(c Dashboard) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		c.render(w, r, http.StatusOK, view{})
	})
	// La feuille de style et le script, derrière l'Accès comme le reste.
	mux.Handle("GET /static/", http.FileServerFS(static))
	mux.HandleFunc("POST /stores/{store}/seuil", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("store")
		// Une valeur vide rend au Store le Seuil par défaut.
		var seuil *time.Duration
		if text := r.FormValue("seuil"); strings.TrimSpace(text) != "" {
			d, err := etat.ParseSeuil(text)
			if err != nil {
				c.render(w, r, http.StatusBadRequest, view{seuil: &seuilRefused{store: name, text: text, err: err.Error()}})
				return
			}
			seuil = &d
		}
		if err := c.Catalogue.SetSeuil(name, seuil); err != nil {
			if errors.Is(err, catalogue.ErrUnknownStore) {
				http.NotFound(w, r)
				return
			}
			c.render(w, r, http.StatusInternalServerError, view{seuil: &seuilRefused{store: name, text: r.FormValue("seuil"), err: err.Error()}})
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
	mux.HandleFunc("POST /stores/{store}/exploration", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("store")
		e, ok := c.Catalogue.Entry(name)
		if !ok {
			http.NotFound(w, r)
			return
		}
		if e.Port == 0 {
			c.render(w, r, http.StatusConflict, view{exploration: &explorationRefused{store: name, err: plageEpuisee}})
			return
		}
		if _, err := c.Explorations.Lancer(name, e.Port); err != nil {
			c.render(w, r, http.StatusBadGateway, view{exploration: &explorationRefused{store: name, err: err.Error()}})
			return
		}
		// La page montre alors le lien de l'Exploration.
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /stores/{store}/exploration/arret", func(w http.ResponseWriter, r *http.Request) {
		c.Explorations.Arreter(r.PathValue("store"))
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /commande", func(w http.ResponseWriter, r *http.Request) {
		// La ligne saisie n'est ni journalisée, ni renvoyée dans la page.
		sortie, err := c.Commande(r.Context(), r.FormValue("commande"))
		res := &commandeResult{Sortie: sortie}
		status := http.StatusOK
		var refus *commande.Refus
		switch {
		case errors.As(err, &refus):
			res.Refus = refus.Error()
			status = http.StatusBadRequest
		case err != nil:
			res.Erreur = err.Error()
			status = http.StatusInternalServerError
		}
		c.render(w, r, status, view{commande: res})
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

// commandeResult est le résultat d'une Commande de Store, à afficher dans son
// bloc : la sortie de plakar, ou le refus de la commande, ou l'erreur qui l'a
// empêchée. Une Sortie peut s'accompagner d'une Erreur : la commande a réussi,
// mais le tableau de bord n'a pas pu être mis à jour.
type commandeResult struct {
	Sortie        *commande.Sortie
	Refus, Erreur string
}

// plageEpuisee remplace le bouton « Explorer » d'un Store sans port.
const plageEpuisee = "Plage de ports d'Exploration épuisée : agrandissez PLAKAR_UI_PORTS."

// explorationRefused est l'échec du lancement d'une Exploration, à signaler
// sur la ligne de son Store.
type explorationRefused struct {
	store, err string
}

// view est ce qu'une réponse ajoute au tableau de bord : un Seuil refusé, un
// lancement d'Exploration en échec, ou le résultat d'une Commande de Store.
type view struct {
	seuil       *seuilRefused
	exploration *explorationRefused
	commande    *commandeResult
}

// render écrit le tableau de bord de l'Utilisateur autorisé de la requête,
// avec ce que v y ajoute.
func (c Dashboard) render(w http.ResponseWriter, r *http.Request, status int, v view) {
	now := c.Now()
	rows := make([]row, 0)
	// Injoignable est le dernier État.
	var parEtat [etat.Injoignable + 1]int
	for _, e := range c.Catalogue.Entries() {
		line := c.newRow(e, now)
		// Un Store sans Relevé terminé n'est compté nulle part.
		if line.etat != nil {
			parEtat[*line.etat]++
		}
		// Le dépliage d'un Store s'ouvre de lui-même sur l'action qui vient d'y
		// échouer.
		if v.seuil != nil && v.seuil.store == e.Name {
			line.SeuilInput, line.SeuilError = v.seuil.text, v.seuil.err
			line.Open = true
		}
		if v.exploration != nil && v.exploration.store == e.Name {
			line.ExplorationError = v.exploration.err
			line.Open = true
		}
		rows = append(rows, line)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// La page porte les jetons des Explorations en cours, et la sortie d'une
	// Commande de Store : rien ne doit en rester dans le cache du navigateur.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	data := struct {
		User      string
		Compteurs []compteur
		Rows      []row
		Commande  *commandeResult
	}{User: acces.User(r.Context()), Compteurs: compteurs(parEtat), Rows: rows, Commande: v.commande}
	if err := page.Execute(w, data); err != nil {
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
	// ReleveHeure est l'heure seule du dernier Relevé terminé, pour la ligne
	// repliée.
	ReleveHeure string
	// Open ouvre le dépliage du Store : une action sur ce Store vient
	// d'échouer.
	Open bool
	// Etat est le mot de l'État, « Premier Relevé » tant qu'aucun Relevé
	// n'est terminé ; Voyant est la couleur qui l'accompagne, vide sans État.
	Etat, Voyant string
	etat         *etat.Etat
	Seuil        string
	// SeuilInput est la valeur du champ de Seuil, vide pour le Seuil par
	// défaut ; SeuilError explique pourquoi une valeur saisie a été refusée.
	SeuilInput, SeuilError string
	Latest                 *snapshot
	// Message remplace le Dernier snapshot quand il n'y en a pas.
	Message string
	// Error est le message d'erreur du dernier Relevé, s'il a échoué.
	Error string
	// Perime : le Store est Injoignable, ses valeurs affichées sont les
	// dernières connues, à atténuer.
	Perime bool
	// Exploration est le lien de l'Exploration en cours, vide sans
	// Exploration ; PortMessage remplace le bouton « Explorer » d'un Store
	// sans port ; ExplorationError explique l'échec d'un lancement.
	Exploration, PortMessage, ExplorationError string
}

// dateLayout est le format des dates affichées, en heure locale.
const dateLayout = "2006-01-02 15:04"

// heureLayout est le format de l'heure seule, en heure locale.
const heureLayout = "15:04"

// snapshot est le Dernier snapshot d'une ligne, mis en forme, avec l'Écart
// (vide sans Snapshot précédent).
type snapshot struct {
	Date, Age, ID, Size, Duration, Path, Tags string
	SizeEcart, DurationEcart                  *ecart
	// Previous décrit le Snapshot précédent, auquel l'Écart compare.
	Previous string
}

func (c Dashboard) newRow(e catalogue.Entry, now time.Time) row {
	seuil := c.DefaultSeuil
	r := row{Name: e.Name, EnCours: e.EnCours, Seuil: etat.FormatSeuil(seuil) + " (par défaut)", Exploration: c.Explorations.Lien(e.Name)}
	if e.Port == 0 {
		r.PortMessage = plageEpuisee
	}
	if e.Seuil != nil {
		seuil = *e.Seuil
		r.Seuil = etat.FormatSeuil(seuil)
		r.SeuilInput = r.Seuil
	}
	r.Etat = "Premier Relevé"
	if e.Releve != nil {
		st := etat.Of(e.Releve.Latest, seuil, now, e.Releve.Err)
		r.Etat, r.Voyant, r.etat = st.String(), voyant(st), &st
		at := e.Releve.At.In(c.Location)
		r.ReleveAt, r.ReleveHeure = at.Format(dateLayout), at.Format(heureLayout)
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
	r.Perime = e.Releve.Err != nil
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
		size := sizeEcart(*ec)
		sign := cmp.Compare(ec.Duration, 0)
		r.Latest.SizeEcart = &size
		r.Latest.DurationEcart = &ecart{Texte: signed(sign, ec.Duration.Abs().String()), Sens: sens(sign)}
	}
	return r
}

// voyant est la couleur d'un État, nommée comme dans la feuille de style.
func voyant(e etat.Etat) string {
	switch e {
	case etat.AJour:
		return "a-jour"
	case etat.EnRetard:
		return "en-retard"
	case etat.Injoignable:
		return "injoignable"
	default:
		panic(fmt.Sprintf("État sans voyant : %d", e))
	}
}

// compteur est un compteur du bandeau : le nombre de Stores d'un État.
type compteur struct {
	Voyant, Etat string
	N            int
}

// compteurs sont les compteurs du bandeau, un par État, depuis le nombre de
// Stores de chaque État.
func compteurs(parEtat [etat.Injoignable + 1]int) []compteur {
	out := make([]compteur, 0, len(parEtat))
	for e, n := range parEtat {
		st := etat.Etat(e)
		out = append(out, compteur{Voyant: voyant(st), Etat: st.String(), N: n})
	}
	return out
}

// ecart est un Écart prêt à afficher : son texte signé et son sens.
type ecart struct {
	// Texte est l'Écart signé : « −97 % », « +4s ».
	Texte string
	// Sens est sa flèche : ↑, ↓ ou →.
	Sens string
}

// sizeEcart écrit l'Écart de taille en pourcentage arrondi : « −97 % ».
func sizeEcart(e releve.Ecart) ecart {
	if e.SizeFromZero {
		return ecart{Texte: "nouveau", Sens: sens(1)}
	}
	pct := math.Round(e.Size * 100)
	sign := cmp.Compare(pct, 0)
	return ecart{Texte: signed(sign, fmt.Sprintf("%.0f\u00a0%%", math.Abs(pct))), Sens: sens(sign)}
}

// sens est la flèche d'un Écart de signe sign (−1, 0 ou 1) : ↓, → ou ↑.
func sens(sign int) string {
	switch sign {
	case -1:
		return "↓"
	case 1:
		return "↑"
	default:
		return "→"
	}
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
