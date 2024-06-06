package azugo

import (
	"azugo.io/azugo/token"

	"go.uber.org/zap"
)

// UserAuthorizer is an interface to check if user is authorized.
type UserAuthorizer interface {
	// Authorized returns if user is authorized.
	Authorized() bool
}

// UserDisplayNamer is an interface that provides method for user display name.
type UserDisplayNamer interface {
	// GivenName returns users given name.
	GivenName() string
	// FamilyName returns users family name.
	FamilyName() string
	// DisplayName returns user display name.
	DisplayName() string
}

// UserGrantedScopes is an interface that provides methods to check user granted scopes.
type UserGrantedScopes interface {
	// HasScopeGroup checks if user has any granted scopes in specified group.
	HasScopeGroup(name string) bool
	// HasScope checks if user has granted scope with any level.
	HasScope(name string) bool
	// HasScopeLevel checks if user has granted scope with exact level.
	HasScopeLevel(name string, level string) bool
	// HasScopeAnyLevel checks if user has granted scope with one of levels.
	HasScopeAnyLevel(name string, levels ...string) bool
}

// UserClaimer is an interface that provides methods to get user claims.
type UserClaimer interface {
	// Claim returns user claim with all values.
	Claim(name ...string) token.ClaimStrings
	// ClaimValue returns user claim with first value.
	ClaimValue(name ...string) string
}

// User is an interface that provides methods to get user information.
type User interface {
	UserAuthorizer
	UserDisplayNamer
	UserGrantedScopes
	UserClaimer

	// ID returns user ID.
	ID() string
}

// SetUser sets authorized user.
func (c *Context) SetUser(u User) {
	c.user = u
	if u != nil && u.Authorized() {
		_ = c.AddLogFields(
			zap.String("user.id", u.ID()),
			zap.String("user.full_name", u.DisplayName()),
		)
	}
}

// User returns authorized user or.
func (c *Context) User() User {
	return c.user
}
