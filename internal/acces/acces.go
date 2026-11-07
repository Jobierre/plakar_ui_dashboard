// Package acces ne laisse passer que les Utilisateurs autorisés.
//
// L'identité est lue dans l'en-tête Tailscale-User-Login, que pose
// `tailscale serve` après avoir retiré toute valeur forgée par le client. Elle
// n'est donc fiable que si le service n'est joignable qu'à travers
// `tailscale serve`, c'est-à-dire s'il n'écoute que sur localhost.
//
// Cela ne suffit pas face au DNS rebinding : une page malveillante ouverte
// dans un navigateur de la même machine peut faire pointer son propre nom vers
// 127.0.0.1, devenir « même origine » que le service et poser elle-même
// l'en-tête d'identité. L'en-tête Host d'une telle requête porte alors le nom
// de l'attaquant : seuls les noms attendus du service sont donc servis.
package acces

import (
	"context"
	"net"
	"net/http"
	"slices"
	"strings"
)

// IdentityHeader est l'en-tête où `tailscale serve` pose l'identité Tailscale
// de l'utilisateur.
const IdentityHeader = "Tailscale-User-Login"

// localHosts sont les noms sous lesquels le service est toujours joignable
// depuis sa propre machine.
var localHosts = []string{"localhost", "127.0.0.1", "::1"}

// Acces est le contrôle d'accès du service.
type Acces struct {
	// Users sont les Utilisateurs autorisés : des identités Tailscale (emails).
	Users []string
	// Hosts sont les noms publics du service (son nom Tailscale), servis en
	// plus des noms de localhost.
	Hosts []string
}

// Handler ne transmet à next que les requêtes adressées à un nom attendu du
// service, puis, parmi elles, que celles d'un Utilisateur autorisé. Il refuse
// les autres avec 403. L'identité est ensuite disponible par User.
func (a Acces) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Le nom d'hôte d'abord : derrière un nom inattendu, l'en-tête
		// d'identité lui-même ne prouve rien.
		if !a.expectedHost(r.Host) {
			http.Error(w, "Accès refusé : nom d'hôte inattendu.", http.StatusForbidden)
			return
		}
		user := r.Header.Get(IdentityHeader)
		if user == "" {
			http.Error(w, "Accès refusé : aucune identité Tailscale.", http.StatusForbidden)
			return
		}
		if !slices.ContainsFunc(a.Users, func(u string) bool { return strings.EqualFold(u, user) }) {
			http.Error(w, "Accès refusé : cette identité Tailscale n'est pas un Utilisateur autorisé.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

// expectedHost dit si l'en-tête Host désigne un nom attendu du service, quel
// que soit le port.
func (a Acces) expectedHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		// Pas de port.
		host = strings.TrimSuffix(strings.TrimPrefix(hostport, "["), "]")
	}
	// « nom. » est la forme absolue de « nom ».
	host = strings.TrimSuffix(host, ".")
	same := func(h string) bool { return strings.EqualFold(h, host) }
	return host != "" && (slices.ContainsFunc(localHosts, same) || slices.ContainsFunc(a.Hosts, same))
}

type userKey struct{}

// User renvoie l'identité de l'Utilisateur autorisé d'une requête qui a passé
// le contrôle d'accès.
func User(ctx context.Context) string {
	user, _ := ctx.Value(userKey{}).(string)
	return user
}
