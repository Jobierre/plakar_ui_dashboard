// Package tailscaleserve écrit la configuration de `tailscale serve` du
// service : le sidecar Tailscale la surveille et l'applique dès qu'elle change.
package tailscaleserve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// certDomain est remplacé par le sidecar Tailscale (containerboot) par le nom
// complet du nœud, plakar-ui.<tailnet>.ts.net.
const certDomain = "${TS_CERT_DOMAIN}"

// config reprend le format JSON d'ipn.ServeConfig.
type config struct {
	TCP map[string]tcpHandler `json:"TCP"`
	Web map[string]webConfig  `json:"Web"`
}

type tcpHandler struct {
	HTTPS bool `json:"HTTPS"`
}

type webConfig struct {
	Handlers map[string]httpHandler `json:"Handlers"`
}

type httpHandler struct {
	Proxy string `json:"Proxy"`
}

// Write écrit dans path la configuration qui sert en HTTPS chaque port public
// de targets, relayé vers l'URL associée : 443 vers le tableau de bord, chaque
// port d'Exploration vers le relais qui écoute pour lui.
//
// L'écriture est atomique : le sidecar s'arrête sur un fichier illisible, il
// ne doit jamais en voir un à moitié écrit.
func Write(path string, targets map[int]string) error {
	c := config{TCP: map[string]tcpHandler{}, Web: map[string]webConfig{}}
	for port, target := range targets {
		p := strconv.Itoa(port)
		c.TCP[p] = tcpHandler{HTTPS: true}
		c.Web[certDomain+":"+p] = webConfig{Handlers: map[string]httpHandler{"/": {Proxy: target}}}
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".serve-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return err
	}
	// Le sidecar tourne sous un autre utilisateur que le service : le fichier
	// doit lui rester lisible.
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
