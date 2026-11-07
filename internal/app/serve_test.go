package app_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// serveConfig reprend le format de `tailscale serve` (ipn.ServeConfig) lu par
// le sidecar Tailscale.
type serveConfig struct {
	TCP map[string]struct{ HTTPS bool }
	Web map[string]struct {
		Handlers map[string]struct{ Proxy string }
	}
}

// readServeConfig lit le fichier écrit par le service, réduit à ce que
// Tailscale en fait : pour chaque port HTTPS, la cible du proxy.
func readServeConfig(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config tailscale serve : %v", err)
	}
	var sc serveConfig
	if err := json.Unmarshal(raw, &sc); err != nil {
		t.Fatalf("config tailscale serve illisible : %v\n%s", err, raw)
	}
	got := map[string]string{}
	for port, h := range sc.TCP {
		if !h.HTTPS {
			t.Errorf("port %s : HTTPS attendu", port)
		}
		web := sc.Web["${TS_CERT_DOMAIN}:"+port]
		if len(web.Handlers) != 1 {
			t.Errorf("port %s : un seul handler « / » attendu, reçu %v", port, web.Handlers)
		}
		got[port] = web.Handlers["/"].Proxy
	}
	if len(sc.Web) != len(sc.TCP) {
		t.Errorf("Web = %v, attendu une entrée par port de TCP", sc.Web)
	}
	return got
}

func TestServiceWritesTailscaleServeConfigForDashboardAndExplorationPorts(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores("")
	path := filepath.Join(t.TempDir(), "serve", "serve.json")
	base := freePorts(3)
	svc := startService(t, plakar, newClock("2026-11-07T17:00:00Z"), map[string]string{
		"PLAKAR_UI_SERVE_CONFIG": path,
		"PLAKAR_UI_PORTS":        fmt.Sprintf("%d-%d", base, base+2),
	})

	got := readServeConfig(t, path)

	want := map[string]string{
		"443": svc.app.URL(),
	}
	for p := base; p <= base+2; p++ {
		want[fmt.Sprint(p)] = fmt.Sprintf("http://127.0.0.1:%d", p)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tailscale serve = %v\nattendu %v", got, want)
	}
}

func TestServiceRewritesTailscaleServeConfigWhenThePortRangeChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.json")
	for _, n := range []int{3, 1} {
		plakar := newFakePlakar(t)
		plakar.stores("")
		base := freePorts(n)
		svc := startService(t, plakar, newClock("2026-11-07T17:00:00Z"), map[string]string{
			"PLAKAR_UI_SERVE_CONFIG": path,
			"PLAKAR_UI_PORTS":        fmt.Sprintf("%d-%d", base, base+n-1),
		})
		svc.stop()

		got := readServeConfig(t, path)
		if len(got) != n+1 || got[fmt.Sprint(base)] == "" {
			t.Errorf("plage de %d ports à partir de %d : tailscale serve = %v", n, base, got)
		}
	}
}
