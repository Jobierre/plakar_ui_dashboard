// Package planificateur déclenche les Relevés des Stores : un par Store à
// intervalle régulier, et à la demande. Deux Relevés d'un même Store ne se
// chevauchent jamais.
package planificateur

import (
	"context"
	"log/slog"
	"slices"
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
	// schedules sont les planifications des Stores suivis. Il faut tenir mu.
	schedules map[string]schedule
}

// schedule est la planification des Relevés d'un Store. ctx est annulé au
// retrait du Store, ce qui coupe aussi son Relevé en cours.
type schedule struct {
	ctx  context.Context
	stop context.CancelFunc
}

// Start lance la planification des Relevés de ces Stores : un premier Relevé
// tout de suite, puis un à chaque Intervalle. Elle s'arrête quand ctx est
// annulé ; Wait attend alors la fin des Relevés en cours. Trigger ne doit
// être appelé qu'après Start.
func (p *Planificateur) Start(ctx context.Context, names []string) {
	p.ctx = ctx
	p.sem = make(chan struct{}, releveParallelism)
	p.schedules = map[string]schedule{}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, name := range names {
		p.startSchedule(name)
	}
}

// SetStores remplace les Stores du Catalogue par ceux-ci, lus dans la config
// plakar : un Store ajouté est relevé tout de suite, puis à chaque Intervalle ;
// un Store retiré n'est plus relevé. L'erreur est celle de l'enregistrement du
// Catalogue, dont la liste est changée malgré tout.
func (p *Planificateur) SetStores(names []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ctx.Err() != nil {
		return nil
	}
	// La planification d'un Store retiré s'arrête avant qu'il ne quitte le
	// Catalogue : elle ne le trouve jamais inconnu.
	for name, s := range p.schedules {
		if !slices.Contains(names, name) {
			s.stop()
			delete(p.schedules, name)
		}
	}
	err := p.Catalogue.SetStores(names)
	for _, name := range names {
		if _, ok := p.schedules[name]; !ok {
			p.startSchedule(name)
		}
	}
	return err
}

// startSchedule commence la planification des Relevés du Store : un premier
// Relevé tout de suite, si bien qu'il est déjà en cours au retour, puis un à
// chaque Intervalle. L'échéance suivante est fixée avant de lancer le Relevé :
// l'Intervalle se compte d'un début de Relevé au suivant, quelle que soit sa
// durée. Il faut tenir mu.
func (p *Planificateur) startSchedule(name string) {
	ctx, stop := context.WithCancel(p.ctx)
	p.schedules[name] = schedule{ctx: ctx, stop: stop}
	next := p.Now().Add(p.Intervalle)
	if err := p.trigger(name); err != nil {
		p.Log.Error("Relevé planifié", "store", name, "err", err)
		stop()
		delete(p.schedules, name)
		return
	}
	p.wg.Go(func() { p.loop(ctx, name, next) })
}

// loop relève le Store à chaque échéance, à partir de next. Une échéance
// qui tombe pendant un Relevé en cours (lancé par ↻) est sautée. Elle s'arrête
// quand ctx est annulé : arrêt du service ou retrait du Store.
func (p *Planificateur) loop(ctx context.Context, name string, next time.Time) {
	for {
		select {
		case <-p.After(next.Sub(p.Now())):
		case <-ctx.Done():
			return
		}
		next = p.Now().Add(p.Intervalle)
		if ctx.Err() != nil {
			return
		}
		if err := p.Trigger(name); err != nil {
			p.Log.Error("Relevé planifié", "store", name, "err", err)
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
	return p.trigger(name)
}

// trigger est Trigger. Il faut tenir mu.
func (p *Planificateur) trigger(name string) error {
	if p.ctx.Err() != nil {
		return nil
	}
	j, started, err := p.Catalogue.Begin(name)
	if err != nil || !started {
		return err
	}
	// Le Relevé vit avec la planification du Store : le retrait du Store le
	// coupe.
	ctx := p.ctx
	if s, ok := p.schedules[name]; ok {
		ctx = s.ctx
	}
	p.wg.Go(func() { p.run(ctx, name, j) })
	return nil
}

// run fait le Relevé commencé du Store et l'enregistre. Le Store n'est ouvert
// que le temps de la lecture : plakar a terminé quand Run rend la main. Un
// Relevé coupé par l'arrêt du service ou le retrait du Store (ctx annulé)
// n'est pas enregistré : ce n'est pas un échec du Store, et il ne doit pas le
// faire paraître Injoignable. Un Relevé qui a abouti juste avant est
// enregistré.
func (p *Planificateur) run(ctx context.Context, name string, j catalogue.Jeton) {
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		p.Catalogue.Abandon(name, j)
		return
	}
	defer func() { <-p.sem }()
	rctx, done := context.WithTimeout(ctx, releveTimeout)
	defer done()
	r := releve.Run(rctx, p.Plakar, name, p.Now)
	if r.Err != nil && ctx.Err() != nil {
		p.Catalogue.Abandon(name, j)
		return
	}
	if r.Err != nil {
		p.Log.Warn("Relevé en échec", "store", name, "err", r.Err)
	}
	if err := p.Catalogue.Record(name, j, r); err != nil {
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
