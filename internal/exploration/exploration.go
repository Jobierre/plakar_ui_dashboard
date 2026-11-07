// Package exploration lance et arrête les Explorations : le `plakar ui`
// officiel sur un Store, que le service relaie sur le port public du Store.
//
// Relayer plutôt qu'exposer `plakar ui` directement permet au service de
// vérifier l'identité de l'utilisateur sur l'Exploration, et de mesurer son
// inactivité. Le jeton de `plakar ui` ne figure jamais dans les journaux.
package exploration

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// startTimeout borne l'attente de la ligne de lancement de `plakar ui` :
	// ouvrir un gros Kloset à froid prend environ une minute.
	startTimeout = 3 * time.Minute
	// launchPrefix ouvre la ligne où `plakar ui` annonce son adresse et son
	// jeton : « launching webUI at http://127.0.0.1:9400?plakar_token=<uuid> ».
	launchPrefix = "launching webUI at "
	// maxOutput borne la sortie gardée pour expliquer un échec de lancement.
	maxOutput = 4096
)

// Explorations sont les Explorations en cours, au plus une par Store.
type Explorations struct {
	// Bin est le binaire plakar.
	Bin string
	// Decalage sépare le port public d'un Store du port interne où écoute son
	// `plakar ui`.
	Decalage int
	// Inactivite est la durée sans trafic après laquelle une Exploration est
	// arrêtée.
	Inactivite time.Duration
	// Hote est le nom public du service, qui fait les liens
	// https://<Hote>:<port>/ ; vide, les liens sont http://localhost:<port>/.
	Hote string
	// Now donne l'heure courante ; After attend que l'horloge ait avancé
	// d'une durée (time.Now et time.After en production).
	Now   func() time.Time
	After func(time.Duration) <-chan time.Time
	Log   *slog.Logger

	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	running map[string]*exploration
	wg      sync.WaitGroup
}

// exploration est une Exploration lancée ou en cours de lancement.
type exploration struct {
	store string
	port  int
	// stop arrête `plakar ui`, même pendant son lancement ; exited est fermé
	// une fois le processus terminé, ou s'il n'a pas pu être lancé.
	stop   context.CancelFunc
	exited chan struct{}
	// ready est fermé une fois le lancement terminé : err dit alors s'il a
	// échoué.
	ready chan struct{}
	err   error
	// prete indique que `plakar ui` a annoncé son jeton : token et proxy
	// sont alors prêts. Il faut tenir Explorations.mu.
	prete bool
	token string
	proxy *httputil.ReverseProxy
	// last est l'heure du dernier trafic, inFlight le nombre de requêtes en
	// cours. Il faut tenir Explorations.mu.
	last     time.Time
	inFlight int
}

// Start prépare les Explorations : elles s'arrêtent toutes quand ctx est
// annulé, ou à Close.
func (e *Explorations) Start(ctx context.Context) {
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.running = map[string]*exploration{}
}

// Lancer lance l'Exploration du Store sur ce port public, attend que
// `plakar ui` soit prêt et renvoie le lien de l'Exploration, jeton compris. Si
// l'Exploration du Store est déjà lancée, ou en cours de lancement, il renvoie
// son lien sans lancer de second processus. Un échec rapporte la sortie de
// plakar. Le port ne doit être tenu par aucune autre Exploration.
func (e *Explorations) Lancer(store string, port int) (string, error) {
	e.mu.Lock()
	if e.ctx.Err() != nil {
		e.mu.Unlock()
		return "", errors.New("le service s'arrête")
	}
	if x, ok := e.running[store]; ok {
		e.mu.Unlock()
		<-x.ready
		if x.err != nil {
			return "", x.err
		}
		return e.link(x), nil
	}
	for _, other := range e.running {
		if other.port == port {
			e.mu.Unlock()
			return "", fmt.Errorf("le port %d est encore tenu par l'Exploration de %s", port, other.store)
		}
	}
	ctx, stop := context.WithCancel(e.ctx)
	x := &exploration{store: store, port: port, stop: stop, exited: make(chan struct{}), ready: make(chan struct{})}
	e.running[store] = x
	// Compté sous le verrou, avant tout Close : Close attend ce lancement.
	e.wg.Add(1)
	e.mu.Unlock()
	defer e.wg.Done()

	x.err = e.start(ctx, x)
	close(x.ready)
	if x.err != nil {
		e.forget(x)
		return "", x.err
	}
	e.Log.Info("Exploration lancée", "store", store, "port", port)
	e.wg.Go(func() { e.watch(x) })
	return e.link(x), nil
}

// start lance `plakar ui` et attend sa ligne de lancement. Annuler ctx
// (x.stop) arrête le processus, et le lancement échoue.
func (e *Explorations) start(ctx context.Context, x *exploration) error {
	internal := fmt.Sprintf("127.0.0.1:%d", x.port+e.Decalage)
	cmd := exec.CommandContext(ctx, e.Bin, "at", "@"+x.store, "ui", "-no-spawn", "-addr", internal)
	// SIGTERM laisse à plakar le temps de refermer le Store proprement.
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 10 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		close(x.exited)
		return err
	}
	var stderr boundedBuffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		close(x.exited)
		return fmt.Errorf("plakar n'a pas pu être lancé : %w", err)
	}

	tokens := make(chan string, 1)
	var other boundedBuffer
	var waitErr error
	e.wg.Go(func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			if token, ok := parseLaunch(sc.Text()); ok {
				select {
				case tokens <- token:
				default:
				}
				continue
			}
			other.Write(append(sc.Bytes(), '\n'))
		}
		// Wait seulement une fois la sortie lue en entier.
		waitErr = cmd.Wait()
		close(x.exited)
	})

	select {
	case token := <-tokens:
		e.mu.Lock()
		defer e.mu.Unlock()
		x.prete, x.token = true, token
		x.proxy = httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: internal})
		x.last = e.Now()
		return nil
	case <-x.exited:
		if ctx.Err() != nil {
			return errors.New("Exploration arrêtée pendant son lancement")
		}
		msg := strings.TrimSpace(stderr.String() + other.String())
		if msg == "" {
			msg = fmt.Sprintf("plakar ui s'est arrêté (%v)", waitErr)
		}
		return errors.New(msg)
	case <-e.After(startTimeout):
		x.stop()
		<-x.exited
		return fmt.Errorf("plakar ui n'a pas annoncé son adresse en %s", startTimeout)
	}
}

// parseLaunch lit le jeton dans la ligne de lancement de `plakar ui`.
func parseLaunch(line string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(line), launchPrefix)
	if !ok {
		return "", false
	}
	u, err := url.Parse(rest)
	if err != nil {
		return "", false
	}
	token := u.Query().Get("plakar_token")
	return token, token != ""
}

// watch arrête l'Exploration après Inactivite sans trafic, et l'oublie quand
// son processus se termine.
func (e *Explorations) watch(x *exploration) {
	for {
		e.mu.Lock()
		wait := e.Inactivite
		if x.inFlight == 0 {
			wait = x.last.Add(e.Inactivite).Sub(e.Now())
		}
		e.mu.Unlock()
		if wait <= 0 {
			e.Log.Info("Exploration arrêtée après inactivité", "store", x.store)
			x.stop()
			<-x.exited
			e.forget(x)
			return
		}
		select {
		case <-e.After(wait):
		case <-x.exited:
			e.forget(x)
			return
		}
	}
}

// forget retire l'Exploration des Explorations en cours.
func (e *Explorations) forget(x *exploration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running[x.store] == x {
		delete(e.running, x.store)
	}
}

// link est le lien public de l'Exploration, jeton compris.
func (e *Explorations) link(x *exploration) string {
	scheme, host := "https", e.Hote
	if host == "" {
		scheme, host = "http", "localhost"
	}
	return fmt.Sprintf("%s://%s:%d/?plakar_token=%s", scheme, host, x.port, url.QueryEscape(x.token))
}

// Lien renvoie le lien de l'Exploration en cours du Store, vide s'il n'y en a
// pas ou si elle est encore en cours de lancement.
func (e *Explorations) Lien(store string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if x, ok := e.running[store]; ok && x.prete {
		return e.link(x)
	}
	return ""
}

// onPort renvoie l'Exploration prête qui tient ce port public, nil s'il n'y en
// a pas.
func (e *Explorations) onPort(port int) *exploration {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, x := range e.running {
		if x.port == port && x.prete {
			return x
		}
	}
	return nil
}

// Arreter arrête l'Exploration du Store, lancée ou en cours de lancement, s'il
// y en a une, et attend la fin de son processus.
func (e *Explorations) Arreter(store string) {
	e.mu.Lock()
	x := e.running[store]
	e.mu.Unlock()
	if x != nil {
		x.stop()
		<-x.exited
		e.forget(x)
	}
}

// Garder arrête les Explorations des Stores absents de names : des Stores
// retirés de la config plakar.
func (e *Explorations) Garder(names []string) {
	e.mu.Lock()
	var gone []string
	for store := range e.running {
		if !slices.Contains(names, store) {
			gone = append(gone, store)
		}
	}
	e.mu.Unlock()
	for _, store := range gone {
		e.Arreter(store)
	}
}

// Handler relaie le port public vers l'Exploration en cours qui l'utilise. Le
// contrôle d'accès doit l'envelopper.
func (e *Explorations) Handler(port int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		x := e.onPort(port)
		if x == nil {
			http.Error(w, "Aucune Exploration en cours sur ce port : lancez-la depuis le tableau de bord.", http.StatusNotFound)
			return
		}
		e.activity(x, 1)
		defer e.activity(x, -1)
		x.proxy.ServeHTTP(w, r)
	})
}

// activity note le début (+1) ou la fin (-1) d'une requête relayée.
func (e *Explorations) activity(x *exploration, delta int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	x.inFlight += delta
	x.last = e.Now()
}

// Close arrête toutes les Explorations et attend la fin de leurs processus.
// L'annulation se fait sous le verrou : passé ce point, aucun lancement ne
// commence, et le compteur ne remonte plus pendant l'attente.
func (e *Explorations) Close() {
	e.mu.Lock()
	e.cancel()
	e.mu.Unlock()
	e.wg.Wait()
}

// boundedBuffer garde le début d'une sortie, jusqu'à maxOutput octets.
type boundedBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if room := maxOutput - len(b.buf); room > 0 {
		b.buf = append(b.buf, p[:min(len(p), room)]...)
	}
	return len(p), nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}
