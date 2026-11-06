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

	"plakar-ui/internal/catalogue"
	"plakar-ui/internal/plakar"
	"plakar-ui/internal/releve"
	"plakar-ui/internal/web"
)

const (
	// releveTimeout borne la durée d'un Relevé : un Store qui ne répond pas
	// devient Injoignable au lieu de bloquer sa ligne.
	releveTimeout = 10 * time.Minute
	// releveParallelism limite le nombre de Relevés simultanés.
	releveParallelism = 4
)

// Options sont les dépendances injectées dans le service.
type Options struct {
	// Getenv lit une variable d'environnement (os.Getenv en production).
	Getenv func(string) string
	// Now donne l'heure courante (time.Now en production).
	Now func() time.Time
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
}

// Start lit la configuration, charge le Catalogue depuis la config plakar,
// sert le tableau de bord et lance un Relevé de chaque Store.
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
	a := &App{
		url: "http://" + ln.Addr().String(),
		server: &http.Server{Handler: web.Handler(web.Dashboard{
			Catalogue:    cat,
			DefaultSeuil: cfg.DefaultSeuil,
			Now:          opts.Now,
			Location:     opts.Location,
			Log:          log,
		})},
		cancel: cancel,
	}
	a.wg.Go(func() {
		if err := a.server.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			log.Error("serveur HTTP", "err", err)
		}
	})
	log.Info("tableau de bord servi", "url", a.url, "stores", len(names))

	sem := make(chan struct{}, releveParallelism)
	for _, name := range names {
		a.wg.Go(func() {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			rctx, done := context.WithTimeout(ctx, releveTimeout)
			defer done()
			r := releve.Run(rctx, p, name, opts.Now)
			if r.Err != nil {
				log.Warn("Relevé en échec", "store", name, "err", r.Err)
			}
			if err := cat.Record(name, r); err != nil {
				log.Error("enregistrement du Relevé", "store", name, "err", err)
			}
		})
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
	return err
}
