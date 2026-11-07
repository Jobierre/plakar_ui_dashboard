package app

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"plakar-ui/internal/catalogue"
	"plakar-ui/internal/etat"
)

// Config est la configuration du service, lue dans des variables
// d'environnement.
type Config struct {
	// Addr est l'adresse d'écoute du tableau de bord (PLAKAR_UI_ADDR). Elle
	// doit être sur localhost : l'identité Tailscale n'est fiable que derrière
	// `tailscale serve`.
	Addr string
	// PlakarBin est le chemin du binaire plakar (PLAKAR_UI_PLAKAR_BIN).
	PlakarBin string
	// DefaultSeuil est le Seuil de retard des Stores qui n'en ont pas de
	// propre (PLAKAR_UI_SEUIL, 26h par défaut).
	DefaultSeuil time.Duration
	// StatePath est le fichier d'état, qui garde les Seuils et le dernier
	// Relevé de chaque Store (PLAKAR_UI_STATE).
	StatePath string
	// Intervalle sépare deux Relevés planifiés d'un même Store
	// (PLAKAR_UI_INTERVALLE, 1h par défaut).
	Intervalle time.Duration
	// Users sont les Utilisateurs autorisés, des identités Tailscale (emails)
	// séparées par des virgules (PLAKAR_UI_UTILISATEURS).
	Users []string
	// Hote est le nom Tailscale du service, par exemple
	// plakar-ui.<tailnet>.ts.net (PLAKAR_UI_HOTE). Les requêtes adressées à
	// un autre nom que lui ou localhost sont refusées.
	Hote string
	// Ports est la plage des ports publics d'Exploration (PLAKAR_UI_PORTS,
	// 9300-9320 par défaut). Le service écoute sur chacun, en localhost.
	Ports catalogue.Plage
	// Decalage sépare le port public d'un Store du port interne de son
	// `plakar ui` (PLAKAR_UI_DECALAGE, 100 par défaut).
	Decalage int
	// Inactivite est la durée sans trafic après laquelle une Exploration est
	// arrêtée (PLAKAR_UI_INACTIVITE, 15m par défaut).
	Inactivite time.Duration
	// ServeConfig est le fichier de configuration de `tailscale serve` que le
	// service écrit au démarrage, à partir de son adresse et de sa plage de
	// ports (PLAKAR_UI_SERVE_CONFIG). Vide, il n'est pas écrit.
	ServeConfig string
}

func loadConfig(getenv func(string) string) (Config, error) {
	c := Config{
		Addr:        orDefault(getenv("PLAKAR_UI_ADDR"), "127.0.0.1:8080"),
		PlakarBin:   orDefault(getenv("PLAKAR_UI_PLAKAR_BIN"), "plakar"),
		StatePath:   orDefault(getenv("PLAKAR_UI_STATE"), "plakar-ui-state.json"),
		Hote:        strings.TrimSpace(getenv("PLAKAR_UI_HOTE")),
		ServeConfig: getenv("PLAKAR_UI_SERVE_CONFIG"),
	}
	for _, u := range strings.Split(getenv("PLAKAR_UI_UTILISATEURS"), ",") {
		if u = strings.TrimSpace(u); u != "" {
			c.Users = append(c.Users, u)
		}
	}
	// « nom. » est la forme absolue de « nom ».
	c.Hote = strings.TrimSuffix(c.Hote, ".")
	if strings.ContainsAny(c.Hote, ":/") {
		return c, fmt.Errorf("PLAKAR_UI_HOTE %q : un nom d'hôte seul est attendu, sans schéma ni port, par exemple plakar-ui.<tailnet>.ts.net", c.Hote)
	}
	if len(c.Users) == 0 {
		return c, errors.New("PLAKAR_UI_UTILISATEURS est vide : sans Utilisateur autorisé, toute requête serait refusée. Indiquez au moins une identité Tailscale, par exemple PLAKAR_UI_UTILISATEURS=moi@example.com (plusieurs identités séparées par des virgules)")
	}
	seuil, err := etat.ParseSeuil(orDefault(getenv("PLAKAR_UI_SEUIL"), "26h"))
	if err != nil {
		return c, fmt.Errorf("PLAKAR_UI_SEUIL : %w", err)
	}
	c.DefaultSeuil = seuil
	text := orDefault(getenv("PLAKAR_UI_INTERVALLE"), "1h")
	c.Intervalle, err = time.ParseDuration(text)
	if err != nil {
		return c, fmt.Errorf("PLAKAR_UI_INTERVALLE : %w", err)
	}
	if c.Intervalle <= 0 {
		return c, fmt.Errorf("PLAKAR_UI_INTERVALLE %q : une durée positive est attendue, par exemple 1h ou 30m", text)
	}
	if c.Ports, err = catalogue.ParsePlage(orDefault(getenv("PLAKAR_UI_PORTS"), "9300-9320")); err != nil {
		return c, fmt.Errorf("PLAKAR_UI_PORTS : %w", err)
	}
	text = orDefault(getenv("PLAKAR_UI_DECALAGE"), "100")
	c.Decalage, err = strconv.Atoi(text)
	// Les ports internes ne doivent recouper ni la plage publique, ni sortir
	// des ports possibles.
	if err != nil || c.Decalage <= c.Ports.Fin-c.Ports.Debut || c.Ports.Fin+c.Decalage > 65535 {
		return c, fmt.Errorf("PLAKAR_UI_DECALAGE %q : un entier supérieur à la largeur de la plage PLAKAR_UI_PORTS (%d), qui garde les ports internes sous 65536, est attendu, par exemple 100", text, c.Ports.Fin-c.Ports.Debut)
	}
	text = orDefault(getenv("PLAKAR_UI_INACTIVITE"), "15m")
	c.Inactivite, err = time.ParseDuration(text)
	if err != nil || c.Inactivite <= 0 {
		return c, fmt.Errorf("PLAKAR_UI_INACTIVITE %q : une durée positive est attendue, par exemple 15m", text)
	}
	host, _, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return c, fmt.Errorf("PLAKAR_UI_ADDR %q : %w", c.Addr, err)
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return c, fmt.Errorf("PLAKAR_UI_ADDR %q : l'adresse doit être sur localhost", c.Addr)
	}
	return c, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
