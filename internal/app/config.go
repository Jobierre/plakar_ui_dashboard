package app

import (
	"fmt"
	"net"
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
}

func loadConfig(getenv func(string) string) (Config, error) {
	c := Config{
		Addr:      orDefault(getenv("PLAKAR_UI_ADDR"), "127.0.0.1:8080"),
		PlakarBin: orDefault(getenv("PLAKAR_UI_PLAKAR_BIN"), "plakar"),
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
