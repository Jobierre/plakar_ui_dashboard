// Package app assemble le service plakar-ui : Catalogue, Relevés et tableau de
// bord.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"plakar-ui/internal/acces"
	"plakar-ui/internal/catalogue"
	"plakar-ui/internal/commande"
	"plakar-ui/internal/plakar"
	"plakar-ui/internal/planificateur"
	"plakar-ui/internal/web"
)

// Options sont les dépendances injectées dans le service.
type Options struct {
	// Getenv lit une variable d'environnement (os.Getenv en production).
	Getenv func(string) string
	// Now donne l'heure courante (time.Now en production).
	Now func() time.Time
	// After attend que l'horloge ait avancé d'une durée (time.After en
	// production).
	After func(time.Duration) <-chan time.Time
	// Location est le fuseau de l'heure locale affichée (time.Local en
	// production).
	Location *time.Location
	// Log reçoit les journaux du service.
	Log io.Writer
}

// App est le service démarré.
type App struct {
	url    string
	server *http.Server
	cancel context.CancelFunc
	wg     sync.WaitGroup
	plan   *planificateur.Planificateur
}

// Start lit la configuration, charge le Catalogue depuis la config plakar,
// sert le tableau de bord et planifie les Relevés de chaque Store : un tout de
// suite, puis un à chaque intervalle.
func Start(ctx context.Context, opts Options) (*App, error) {
	log := slog.New(slog.NewTextHandler(opts.Log, nil))
	cfg, err := loadConfig(opts.Getenv)
	if err != nil {
		return nil, err
	}
	p := plakar.Plakar{Bin: cfg.PlakarBin}

	names, err := p.Stores(ctx)
	if err != nil {
		return nil, fmt.Errorf("liste des Stores : %w", err)
	}
	cat, err := catalogue.Open(cfg.StatePath, names)
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	plan := &planificateur.Planificateur{
		Plakar:     p,
		Catalogue:  cat,
		Intervalle: cfg.Intervalle,
		Now:        opts.Now,
		After:      opts.After,
		Log:        log,
	}
	executeur := &commande.Executeur{
		Plakar: p,
		Rafraichir: func(ctx context.Context) error {
			names, err := p.Stores(ctx)
			if err != nil {
				return fmt.Errorf("liste des Stores : %w", err)
			}
			return plan.SetStores(names)
		},
	}
	access := acces.Acces{Users: cfg.Users}
	if cfg.Hote != "" {
		access.Hosts = []string{cfg.Hote}
	}
	a := &App{
		url: "http://" + ln.Addr().String(),
		// Le contrôle d'accès enveloppe tout le service : chaque route, y
		// compris celles ajoutées plus tard, passe par lui.
		server: &http.Server{Handler: access.Handler(web.Handler(web.Dashboard{
			Catalogue:    cat,
			DefaultSeuil: cfg.DefaultSeuil,
			Trigger:      plan.Trigger,
			Commande:     executeur.Run,
			Now:          opts.Now,
			Location:     opts.Location,
			Log:          log,
		}))},
		cancel: cancel,
		plan:   plan,
	}
	// La planification démarre avant le serveur : un ↻ trouve toujours le
	// Planificateur prêt.
	plan.Start(ctx, names)
	a.wg.Go(func() {
		if err := a.server.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			log.Error("serveur HTTP", "err", err)
		}
	})
	log.Info("tableau de bord servi", "url", a.url, "stores", len(names), "hote", cfg.Hote)
	if cfg.Hote == "" {
		// Derrière `tailscale serve`, le Host reçu est le nom Tailscale du
		// service : sans lui, toute requête venue du tailnet est refusée.
		log.Warn("PLAKAR_UI_HOTE est vide : seuls les noms de localhost sont servis")
	}

	return a, nil
}

// URL est l'adresse de base du tableau de bord.
func (a *App) URL() string { return a.url }

// Close arrête le service et attend la fin des Relevés en cours.
func (a *App) Close() error {
	a.cancel()
	err := a.server.Close()
	a.wg.Wait()
	a.plan.Wait()
	return err
}
