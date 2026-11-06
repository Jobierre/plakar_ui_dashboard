package app_test

// Test de contrat : le service relève un vrai Kloset avec le vrai plakar. Il
// échoue si une montée de version de plakar change le format de ses sorties.
// Il est sauté quand plakar n'est pas installé ; PLAKAR_UI_CONTRACT_PLAKAR
// désigne au besoin un autre binaire que celui du PATH.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestContractWithRealPlakar(t *testing.T) {
	bin := realPlakar(t)
	k := newKloset(t, bin)

	src := filepath.Join(k.dir, "src")
	writeFile(t, filepath.Join(src, "garde.txt"), 1024)
	writeFile(t, filepath.Join(src, "part.bin"), 3072)
	previous := k.backup(src) // 4 KiB
	if err := os.Remove(filepath.Join(src, "part.bin")); err != nil {
		t.Fatal(err)
	}
	latest := k.backup(src) // 1 KiB

	svc := startServiceWith(t, k.plakar, newClock(time.Now().UTC().Format(time.RFC3339)), nil)
	_, rows := svc.dashboard()

	assertRow(t, rows["contrat"], row{
		"id":         latest,
		"size":       "1.0 KiB",
		"size-ecart": "−75 %",
		"path":       src,
		"tags":       "contrat",
		"error":      "",
	})
	// La durée des backups dépend de la machine : seule sa forme est vérifiée.
	if got := rows["contrat"]["duration-ecart"]; !signedDuration.MatchString(got) {
		t.Errorf("Écart de durée = %q, attendu une durée signée", got)
	}
	want := regexp.MustCompile(`^par rapport au Snapshot précédent ` + previous + ` \(4\.0 KiB, \S+\)$`)
	for _, field := range []string{"size-ecart.title", "duration-ecart.title"} {
		if got := rows["contrat"][field]; !want.MatchString(got) {
			t.Errorf("%s = %q, attendu %v", field, got, want)
		}
	}
}

// signedDuration est un Écart de durée : « 0s », ou une durée précédée de son
// signe.
var signedDuration = regexp.MustCompile(`^(0s|[+−]([0-9]+h)?([0-9]+m)?[0-9.]+(s|ms|µs))$`)

// realPlakar renvoie le vrai binaire plakar, ou saute le test.
func realPlakar(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("PLAKAR_UI_CONTRACT_PLAKAR")
	if bin == "" {
		bin = "plakar"
	}
	path, err := exec.LookPath(bin)
	if err != nil {
		t.Skipf("test de contrat sauté : plakar introuvable (%v)", err)
	}
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		t.Fatalf("plakar version : %v", err)
	}
	t.Logf("test de contrat avec %s (%s)", path, strings.TrimSpace(string(out)))
	return path
}

// kloset est un Kloset jetable, déclaré comme Store @contrat dans une config
// plakar propre au test.
type kloset struct {
	t   *testing.T
	dir string
	// plakar lance le vrai plakar sur la config, le cache et les données du
	// test, sans toucher à ceux de l'utilisateur.
	plakar string
}

func newKloset(t *testing.T, bin string) *kloset {
	t.Helper()
	dir := t.TempDir()
	// plakar ouvre le socket Unix de son démon cached dans le dossier de
	// cache : son chemin doit rester sous la limite des sockets (104 octets
	// sur macOS), que dépasse t.TempDir().
	cache, err := os.MkdirTemp("/tmp", "plakar-ui-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(cache) })
	wrapper := filepath.Join(dir, "plakar")
	script := fmt.Sprintf("#!/bin/sh\nexec %q -configdir %q -cachedir %q -datadir %q \"$@\"\n",
		bin, filepath.Join(dir, "config"), cache, filepath.Join(dir, "data"))
	if err := os.WriteFile(wrapper, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	k := &kloset{t: t, dir: dir, plakar: wrapper}
	store := filepath.Join(dir, "kloset")
	k.run([]string{"PLAKAR_PASSPHRASE=contrat"}, "at", store, "create")
	k.run(nil, "store", "add", "contrat", store, "passphrase=contrat")
	return k
}

// backupID lit l'ID court du snapshot dans la sortie de `plakar backup`.
var backupID = regexp.MustCompile(`(?m)^([0-9a-f]{8}): backup completed`)

// backup sauvegarde ce dossier dans le Kloset et renvoie l'ID court du
// snapshot. Il attend la seconde suivante : plakar date les snapshots à la
// seconde, et deux snapshots de la même seconde n'auraient pas d'ordre sûr.
func (k *kloset) backup(src string) string {
	k.t.Helper()
	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))
	out := k.run(nil, "at", "@contrat", "backup", "-tag", "contrat", src)
	m := backupID.FindStringSubmatch(out)
	if m == nil {
		k.t.Fatalf("ID du snapshot introuvable dans la sortie de backup :\n%s", out)
	}
	return m[1]
}

func (k *kloset) run(env []string, args ...string) string {
	k.t.Helper()
	cmd := exec.Command(k.plakar, args...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		k.t.Fatalf("plakar %s : %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", size)), 0o600); err != nil {
		t.Fatal(err)
	}
}
