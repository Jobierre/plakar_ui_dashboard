// Package planificateur déclenche les Relevés des Stores : un par Store à
// intervalle régulier, et à la demande. Deux Relevés d'un même Store ne se
// chevauchent jamais.
package planificateur

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"plakar-ui/internal/catalogue"
	"plakar-ui/internal/plakar"
	"plakar-ui/internal/releve"
)

const (
	// releveTimeout borne la durée d'un Relevé : un Store qui ne répond pas
	// devient Injoignable au lieu de bloquer sa ligne.
	releveTimeout = 10 * time.Minute
	// releveParallelism limite le nombre de Relevés simultanés, tous Stores
	// confondus.
	releveParallelism = 4
)

// Planificateur fait les Relevés des Stores du Catalogue et y enregistre leur
// résultat.
type Planificateur struct {
	Plakar    plakar.Plakar
	Catalogue *catalogue.Catalogue
	// Intervalle sépare deux Relevés planifiés d'un même Store.
	Intervalle time.Duration
	// Now donne l'heure courante ; After attend que l'horloge ait avancé
	// d'une durée (time.Now et time.After en production).
	Now   func() time.Time
	After func(time.Duration) <-chan time.Time
	Log   *slog.Logger

	ctx context.Context
	sem chan struct{}
	// mu ordonne les lancements de Relevés (Trigger) et l'attente de leur fin
	// (Wait) : aucun Relevé n'est lancé une fois l'arrêt commencé.
	mu sync.Mutex
	wg sync.WaitGroup
}

// Start lance la planification des Relevés de ces Stores : un premier Relevé
// tout de suite, puis un à chaque Intervalle. Elle s'arrête quand ctx est
// annulé ; Wait attend alors la fin des Relevés en cours. Trigger ne doit
// être appelé qu'après Start.
func (p *Planificateur) Start(ctx context.Context, names []string) {
	p.ctx = ctx
	p.sem = make(chan struct{}, releveParallelism)
	for _, name := range names {
		p.wg.Go(func() { p.schedule(name) })
	}
}

// schedule relève le Store à chaque Intervalle. L'échéance suivante est fixée
// avant de lancer le Relevé : l'Intervalle se compte d'un début de Relevé au
// suivant, quelle que soit sa durée. Une échéance qui tombe pendant un Relevé
// en cours (lancé par ↻) est sautée.
func (p *Planificateur) schedule(name string) {
	for {
		next := p.Now().Add(p.Intervalle)
		if err := p.Trigger(name); err != nil {
			p.Log.Error("Relevé planifié", "store", name, "err", err)
			return
		}
		select {
		case <-p.After(next.Sub(p.Now())):
		case <-p.ctx.Done():
			return
		}
	}
}

// Trigger lance tout de suite un Relevé du Store, sans attendre sa fin. Si un
// Relevé de ce Store est déjà en cours, il n'en lance pas de second. L'erreur
// est catalogue.ErrUnknownStore pour un Store absent du Catalogue.
func (p *Planificateur) Trigger(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctx.Err() != nil {
		return nil
	}
	started, err := p.Catalogue.Begin(name)
	if err != nil || !started {
		return err
	}
	p.wg.Go(func() { p.run(name) })
	return nil
}

// run fait le Relevé commencé du Store et l'enregistre. Le Store n'est ouvert
// que le temps de la lecture : plakar a terminé quand Run rend la main. Un
// Relevé coupé par l'arrêt du service n'est pas enregistré : ce n'est pas un
// échec du Store, et il ne doit pas le faire paraître Injoignable au
// redémarrage. Un Relevé qui a abouti juste avant l'arrêt est enregistré.
func (p *Planificateur) run(name string) {
	select {
	case p.sem <- struct{}{}:
	case <-p.ctx.Done():
		p.Catalogue.Abandon(name)
		return
	}
	defer func() { <-p.sem }()
	ctx, done := context.WithTimeout(p.ctx, releveTimeout)
	defer done()
	r := releve.Run(ctx, p.Plakar, name, p.Now)
	if r.Err != nil && p.ctx.Err() != nil {
		p.Catalogue.Abandon(name)
		return
	}
	if r.Err != nil {
		p.Log.Warn("Relevé en échec", "store", name, "err", r.Err)
	}
	if err := p.Catalogue.Record(name, r); err != nil {
		p.Log.Error("enregistrement du Relevé", "store", name, "err", err)
	}
}

// Wait attend la fin des Relevés en cours. ctx doit être annulé : passé le
// verrou, plus aucun Trigger ne lance de Relevé, et le compteur ne peut plus
// remonter pendant l'attente.
func (p *Planificateur) Wait() {
	p.mu.Lock()
	p.mu.Unlock() // section vide voulue : barrière avec Trigger
	p.wg.Wait()
}
