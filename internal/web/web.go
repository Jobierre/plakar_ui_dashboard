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
	"net/url"
	"slices"
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
	// chemin échappe un nom de Store ou de Groupe pour un segment d'adresse :
	// un « / » dans le nom n'y ouvre pas un nouveau segment.
	"chemin": url.PathEscape,
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
	// TriggerAll lance un Relevé immédiat de chacun de ces Stores (bouton
	// « Tout relever »), en sautant ceux dont un Relevé est déjà en cours et
	// ceux sortis du Catalogue entre-temps.
	TriggerAll func(stores []string) error
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
		// ?groupe=<nom> ouvre le focus d'un Groupe, ou de « Sans groupe »
		// (catalogue.SansGroupe).
		c.render(w, r, http.StatusOK, view{accueil: true, focus: r.URL.Query().Get("groupe")})
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
	mux.HandleFunc("POST /releve", func(w http.ResponseWriter, r *http.Request) {
		// Sans Groupe, « Tout relever » relève chaque Store ; avec un Groupe
		// (le focus), « Relever le groupe » ne relève que les siens.
		groupe, retour := r.FormValue("groupe"), "/"
		var names []string
		if groupe == "" {
			for _, e := range c.Catalogue.Entries() {
				names = append(names, e.Name)
			}
		} else {
			var err error
			if names, err = c.Catalogue.Membres(groupe); err != nil {
				c.refuserRangement(w, r, err, cibleRangement{Groupe: groupe}, &rangementRefused{})
				return
			}
			retour = "/?groupe=" + url.QueryEscape(groupe)
		}
		if err := c.TriggerAll(names); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// La page montre alors les Relevés en cours, et se rafraîchit seule.
		http.Redirect(w, r, retour, http.StatusSeeOther)
	})
	mux.HandleFunc("POST /groupes", func(w http.ResponseWriter, r *http.Request) {
		nom := r.FormValue("nom")
		if err := c.Catalogue.CreerGroupe(nom); err != nil {
			c.refuserRangement(w, r, err, cibleRangement{}, &rangementRefused{Nom: nom, Creation: true})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /groupes/{groupe}/nom", func(w http.ResponseWriter, r *http.Request) {
		groupe, nom := r.PathValue("groupe"), r.FormValue("nom")
		if err := c.Catalogue.RenommerGroupe(groupe, nom); err != nil {
			c.refuserRangement(w, r, err, cibleRangement{Groupe: groupe}, &rangementRefused{Nom: nom, Renommage: groupe})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /groupes/{groupe}/suppression", func(w http.ResponseWriter, r *http.Request) {
		groupe := r.PathValue("groupe")
		if err := c.Catalogue.SupprimerGroupe(groupe); err != nil {
			c.refuserRangement(w, r, err, cibleRangement{Groupe: groupe}, &rangementRefused{})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /groupes/{groupe}/ordre", func(w http.ResponseWriter, r *http.Request) {
		cible := cibleRangement{Groupe: r.PathValue("groupe")}
		// Un champ « avant » place le Groupe avant ce Groupe, ou en dernier
		// s'il est vide (le glisser-déposer) ; sinon, « sens » le déplace
		// d'une place (↑/↓).
		var err error
		if avant, ok := champ(r, "avant"); ok {
			cible.Avant = avant
			err = c.Catalogue.PlacerGroupe(cible.Groupe, avant)
		} else if sens, ok := sensDe(r.FormValue("sens")); ok {
			err = c.Catalogue.DeplacerGroupe(cible.Groupe, sens)
		} else {
			err = errSensInconnu
		}
		if err != nil {
			c.refuserRangement(w, r, err, cible, &rangementRefused{})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /stores/{store}/groupe", func(w http.ResponseWriter, r *http.Request) {
		cible := cibleRangement{Store: r.PathValue("store"), Groupe: r.FormValue("groupe")}
		// Un champ « avant » place le Store avant ce Store, ou à la fin s'il
		// est vide (le glisser-déposer). Sans lui (la liste déroulante), un
		// Store déjà dans ce Groupe garde sa place.
		ranger := func() error { return c.Catalogue.Ranger(cible.Store, cible.Groupe) }
		if avant, ok := champ(r, "avant"); ok {
			cible.Avant = avant
			ranger = func() error { return c.Catalogue.Placer(cible.Store, cible.Groupe, avant) }
		}
		if err := ranger(); err != nil {
			c.refuserRangement(w, r, err, cible, &rangementRefused{})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /stores/{store}/ordre", func(w http.ResponseWriter, r *http.Request) {
		store := r.PathValue("store")
		err := errSensInconnu
		if sens, ok := sensDe(r.FormValue("sens")); ok {
			err = c.Catalogue.DeplacerStore(store, sens)
		}
		if err != nil {
			c.refuserRangement(w, r, err, cibleRangement{Store: store}, &rangementRefused{})
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /stores/{store}/retrait", func(w http.ResponseWriter, r *http.Request) {
		store := r.PathValue("store")
		if err := c.Catalogue.RetirerDisparu(store); err != nil {
			c.refuserRangement(w, r, err, cibleRangement{Store: store}, &rangementRefused{})
			return
		}
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

// cibleRangement est ce que vise une action de rangement : un Store, un
// Groupe (ou SansGroupe), et le Store ou le Groupe avant lequel placer.
type cibleRangement struct {
	Store, Groupe, Avant string
}

// refuserRangement renvoie la page avec le refus d'une action de rangement
// sur cette cible ; le dépliage du Store visé s'ouvre.
func (c Dashboard) refuserRangement(w http.ResponseWriter, r *http.Request, err error, cible cibleRangement, refus *rangementRefused) {
	status, msg := expliquer(err, cible)
	refus.Err, refus.Store = msg, cible.Store
	c.render(w, r, status, view{rangement: refus})
}

// expliquer donne le code et le message du refus d'une action de rangement
// sur cette cible : un nom refusé, « Sans groupe » visé, un sens inconnu ou
// un Store de la config visé comme Store disparu sont la faute de la requête
// (400), un Groupe ou un Store inconnu aussi
// (404), un autre échec (l'enregistrement) celle du service (500).
func expliquer(err error, cible cibleRangement) (int, string) {
	affiche := func(groupe string) string {
		if groupe == catalogue.SansGroupe {
			return sansGroupeNom
		}
		return groupe
	}
	var nom *catalogue.NomRefuse
	switch {
	case errors.As(err, &nom), errors.Is(err, errSensInconnu), errors.Is(err, catalogue.ErrSansGroupeFixe):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, catalogue.ErrPasDisparu):
		return http.StatusBadRequest, fmt.Sprintf("Le Store « %s » est dans la config Plakar : il se range, il ne se retire pas.", cible.Store)
	case errors.Is(err, catalogue.ErrPositionInconnue) && cible.Store != "":
		return http.StatusNotFound, fmt.Sprintf("Le Store « %s » n'est pas dans « %s ».", cible.Avant, affiche(cible.Groupe))
	case errors.Is(err, catalogue.ErrPositionInconnue):
		return http.StatusNotFound, fmt.Sprintf("Le Groupe « %s » n'existe pas.", cible.Avant)
	case errors.Is(err, catalogue.ErrUnknownGroupe):
		return http.StatusNotFound, fmt.Sprintf("Le Groupe « %s » n'existe pas.", cible.Groupe)
	case errors.Is(err, catalogue.ErrUnknownStore):
		return http.StatusNotFound, fmt.Sprintf("Le Store « %s » n'existe pas.", cible.Store)
	}
	return http.StatusInternalServerError, err.Error()
}

// champ renvoie la valeur d'un champ du formulaire, et s'il est présent, même
// vide.
func champ(r *http.Request, nom string) (string, bool) {
	v := r.FormValue(nom) // analyse le formulaire
	_, ok := r.Form[nom]
	return v, ok
}

// errSensInconnu refuse un déplacement ni vers le haut, ni vers le bas.
var errSensInconnu = errors.New("Sens de déplacement inconnu.")

// sensDe lit le sens d'un déplacement : « haut » (↑) ou « bas » (↓).
func sensDe(texte string) (catalogue.Sens, bool) {
	switch texte {
	case "haut":
		return catalogue.Haut, true
	case "bas":
		return catalogue.Bas, true
	}
	return 0, false
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
// accueil marque la page d'accueil elle-même, et non la réponse directe à un
// POST : seule elle peut se recharger sans perdre ce qu'elle affiche.
type view struct {
	accueil bool
	// focus désigne le seul Groupe à afficher (ou catalogue.SansGroupe),
	// vide pour tous les Groupes.
	focus       string
	seuil       *seuilRefused
	exploration *explorationRefused
	rangement   *rangementRefused
	commande    *commandeResult
}

// rangementRefused est une action de rangement refusée : son message, et le
// nom de Groupe refusé s'il s'agit d'une création ou d'un renommage.
type rangementRefused struct {
	Err, Nom string
	// Creation : le refus porte sur « + Groupe », dont le formulaire s'ouvre.
	Creation bool
	// Store désigne le Store visé, dont le dépliage s'ouvre.
	Store string
	// Renommage désigne le Groupe dont le renommage est refusé : son menu ⋯
	// s'ouvre, avec le nom refusé.
	Renommage string
}

// sansGroupeNom est le nom affiché de « Sans groupe ».
const sansGroupeNom = "Sans groupe"

// section est un Groupe de la page, ou « Sans groupe » : son en-tête et ses
// Stores. Cle est ce qui le désigne dans les formulaires et dans la page.
type section struct {
	Cle, Nom string
	// Voyant est celui du pire État de ses Stores, vide sans État.
	Voyant    string
	Compteurs []compteur
	Rows      []row
	// Fixe marque « Sans groupe », qui ne se renomme, ne se supprime ni ne
	// se déplace. Premier et Dernier désactivent ↑ et ↓.
	Fixe, Premier, Dernier bool
	// MenuOuvert ouvre le menu ⋯ sur un renommage refusé, avec NomSaisi, le
	// nom refusé, dans son champ.
	MenuOuvert bool
	NomSaisi   string
	// parEtat compte ses Stores par État, enCours ceux dont un Relevé est en
	// cours : le bandeau les additionne sur les Groupes affichés.
	parEtat [etat.Injoignable + 1]int
	enCours int
	// disparus compte ses Stores disparus.
	disparus int
}

// choixGroupe est un rangement proposé par la liste déroulante « Groupe ».
type choixGroupe struct {
	Cle, Nom string
}

// pire est le voyant du pire État parmi ces nombres de Stores par État : En
// retard, puis Injoignable, puis À jour. Il est vide si aucun Store n'a
// d'État.
func pire(parEtat [etat.Injoignable + 1]int) string {
	for _, st := range []etat.Etat{etat.EnRetard, etat.Injoignable, etat.AJour} {
		if parEtat[st] > 0 {
			return voyant(st)
		}
	}
	return ""
}

// render écrit le tableau de bord de l'Utilisateur autorisé de la requête,
// avec ce que v y ajoute.
func (c Dashboard) render(w http.ResponseWriter, r *http.Request, status int, v view) {
	now := c.Now()
	// index numérote les Stores de la page, pour nommer leurs formulaires.
	index := 0
	newRows := func(s *section, entries []catalogue.Entry) {
		s.Rows = make([]row, 0, len(entries))
		for _, e := range entries {
			if e.Disparu {
				// Ni État, ni valeurs : seulement sa place, qu'il peut changer
				// par ↑/↓, et le ménage.
				s.Rows = append(s.Rows, row{Name: e.Name, Index: index, Disparu: true})
				index++
				s.disparus++
				continue
			}
			line := c.newRow(e, now)
			line.Index = index
			index++
			if line.EnCours {
				s.enCours++
			}
			// Un Store sans Relevé terminé n'est compté nulle part.
			if line.etat != nil {
				s.parEtat[*line.etat]++
			}
			// Le dépliage d'un Store s'ouvre de lui-même sur l'action qui vient
			// d'y échouer.
			if v.seuil != nil && v.seuil.store == e.Name {
				line.SeuilInput, line.SeuilError = v.seuil.text, v.seuil.err
				line.Open = true
			}
			if v.rangement != nil && v.rangement.Store == e.Name {
				line.Open = true
			}
			if v.exploration != nil && v.exploration.store == e.Name {
				line.ExplorationError = v.exploration.err
				line.Open = true
			}
			s.Rows = append(s.Rows, line)
		}
	}
	groupes, sans := c.Catalogue.Groupes()
	// choix sont les rangements proposés par la liste déroulante « Groupe » :
	// chaque Groupe, puis « Sans groupe ».
	choix := make([]choixGroupe, 0, len(groupes)+1)
	for _, g := range groupes {
		choix = append(choix, choixGroupe{Cle: g.Nom, Nom: g.Nom})
	}
	choix = append(choix, choixGroupe{Cle: catalogue.SansGroupe, Nom: sansGroupeNom})
	sections := make([]section, 0, len(groupes)+1)
	add := func(cle, nom string, entries []catalogue.Entry) {
		s := section{Cle: cle, Nom: nom, NomSaisi: nom}
		newRows(&s, entries)
		rows := s.Rows
		for i := range rows {
			rows[i].Choix = choix
			rows[i].Groupe = cle
		}
		if len(rows) > 0 {
			rows[0].Premier, rows[len(rows)-1].Dernier = true, true
		}
		s.Voyant, s.Compteurs = pire(s.parEtat), compteurs(s.parEtat)
		if v.rangement != nil && v.rangement.Renommage == cle {
			s.MenuOuvert, s.NomSaisi = true, v.rangement.Nom
		}
		sections = append(sections, s)
	}
	for i, g := range groupes {
		add(g.Nom, g.Nom, g.Stores)
		sections[i].Premier, sections[i].Dernier = i == 0, i == len(groupes)-1
	}
	// « Sans groupe » vient en dernier, et seulement s'il a des Stores, ou
	// que son focus est ouvert.
	if len(sans) > 0 || v.focus == catalogue.SansGroupe {
		add(catalogue.SansGroupe, sansGroupeNom, sans)
		sections[len(sections)-1].Fixe = true
	}
	// En focus, seul le Groupe visé s'affiche, s'il existe.
	inconnu := false
	if v.focus != "" {
		i := slices.IndexFunc(sections, func(s section) bool { return s.Cle == v.focus })
		inconnu = i < 0
		if inconnu {
			sections = nil
		} else {
			sections = sections[i : i+1]
		}
	}
	// Le bandeau ne compte que les Stores affichés.
	var parEtat [etat.Injoignable + 1]int
	// total ne compte que les Stores qu'un Relevé peut viser : pas les
	// Stores disparus.
	enCours, total, disparus := 0, 0, 0
	for _, s := range sections {
		for st, n := range s.parEtat {
			parEtat[st] += n
		}
		enCours += s.enCours
		total += len(s.Rows) - s.disparus
		disparus += s.disparus
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// La page porte les jetons des Explorations en cours, et la sortie d'une
	// Commande de Store : rien ne doit en rester dans le cache du navigateur.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	data := struct {
		User      string
		Compteurs []compteur
		// Disparus compte les Stores disparus affichés : le ménage à faire,
		// affiché même à 0.
		Disparus int
		// ToutEnCours : chaque Store a déjà un Relevé en cours, « Tout
		// relever » n'aurait rien à lancer.
		ToutEnCours bool
		// Rafraichir : au moins un Relevé est en cours, la page d'accueil se
		// recharge seule pour en montrer le résultat. La réponse à un POST
		// (refus, erreur, sortie de Commande) ne se recharge pas : elle
		// perdrait son message.
		Rafraichir bool
		Groupes    []section
		// Rangement est l'échec d'une action de rangement, avec le nom refusé
		// à rendre au formulaire « + Groupe ».
		Rangement *rangementRefused
		Commande  *commandeResult
		// Focus désigne le Groupe en focus, vide pour tous les Groupes ;
		// FocusInconnu : il n'existe pas (renommé, supprimé, adresse fausse).
		Focus        string
		FocusInconnu bool
	}{
		User:         acces.User(r.Context()),
		Compteurs:    compteurs(parEtat),
		Disparus:     disparus,
		ToutEnCours:  enCours == total,
		Rafraichir:   v.accueil && enCours > 0,
		Groupes:      sections,
		Rangement:    v.rangement,
		Commande:     v.commande,
		Focus:        v.focus,
		FocusInconnu: inconnu,
	}
	if err := page.Execute(w, data); err != nil {
		c.Log.Error("rendu du tableau de bord", "err", err)
	}
}

// row est une ligne du tableau de bord, prête à afficher.
type row struct {
	Name string
	// Disparu marque un Store disparu, qui n'a que son nom et sa place.
	Disparu bool
	// Index numérote le Store dans la page, pour nommer ses formulaires.
	Index   int
	Pending bool
	// Groupe désigne le rangement du Store, Choix ceux qu'on peut lui donner.
	// Premier et Dernier désactivent ↑ et ↓ dans son Groupe.
	Groupe           string
	Premier, Dernier bool
	Choix            []choixGroupe
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
