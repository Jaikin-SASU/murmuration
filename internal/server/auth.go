package server

import (
	"net/http"
	"strings"
)

// Authenticator protège l'API publique. L'implémentation par défaut s'appuie sur
// un token statique ; une implémentation adossée à l'Active Directory
// (LDAP/Kerberos) viendra derrière la même interface (ADR-0004).
type Authenticator interface {
	Authenticate(r *http.Request) error
}

// AllowAll n'impose aucune authentification (dev/LAN de confiance uniquement).
type AllowAll struct{}

func (AllowAll) Authenticate(*http.Request) error { return nil }

// StaticToken exige un Bearer token précis.
type StaticToken struct{ Token string }

func (s StaticToken) Authenticate(r *http.Request) error {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) || strings.TrimPrefix(h, prefix) != s.Token {
		return errUnauthorized
	}
	return nil
}

type authError struct{ msg string }

func (e authError) Error() string { return e.msg }

var errUnauthorized = authError{"token invalide ou manquant"}
