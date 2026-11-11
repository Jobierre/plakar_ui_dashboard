package app_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// Vue simple et bascule : le script de l'interface, et la vue par défaut.

func TestScriptIsServedToAnAuthorizedUserOnly(t *testing.T) {
	svc, _ := oneStore(t, nil)
	_, body := svc.get("/")
	script := findElement(t, body, func(n *html.Node) bool {
		_, ok := attr(n, "src")
		return n.Data == "script" && ok
	})
	if script == nil {
		t.Fatalf("la page ne charge aucun script :\n%s", body)
	}
	src, _ := attr(script, "src")

	js := assertServedToAuthorizedOnly(t, svc, src, "text/javascript")
	// Un nom de Store ou de Groupe n'est jamais inséré comme du HTML.
	if strings.Contains(js, "innerHTML") || strings.Contains(js, "insertAdjacentHTML") {
		t.Errorf("le script insère du HTML : il doit manipuler les noms par textContent")
	}
}

func TestSimpleViewIsTheDefaultWithoutJavaScript(t *testing.T) {
	svc, _ := oneStore(t, nil)

	_, body := svc.get("/")

	root := findElement(t, body, func(n *html.Node) bool { return n.Data == "html" })
	class, _ := attr(root, "class")
	if !strings.Contains(" "+class+" ", " vue-simple ") {
		t.Errorf("<html class=%q> : la vue simple doit être la vue par défaut", class)
	}
}
