package user

import (
	"strings"

	"azugo.io/azugo/token"

	"github.com/valyala/bytebufferpool"
)

// Option represents basic user behavior option.
type Option interface {
	apply(u *Basic)
}

// ScopeGroupSeparator is group separator to use when parsing granted scopes.
type ScopeGroupSeparator string

func (s ScopeGroupSeparator) apply(u *Basic) {
	u.scopeGroupSeparator = string(s)
}

// ScopeLevelSeparator is level separator to use when parsing granted scopes.
type ScopeLevelSeparator string

func (s ScopeLevelSeparator) apply(u *Basic) {
	u.scopeLevelSeparator = string(s)
}

// ScopeClaimName is name of claim that contains user granted scopes.
type ScopeClaimName string

func (s ScopeClaimName) apply(u *Basic) {
	u.scopeClaimName = []string{string(s)}
}

// Basic represents authorized user.
type Basic struct {
	scopeSeparator      string
	scopeLevelSeparator string
	scopeGroupSeparator string
	scopeClaimName      []string
	claims              map[string]token.ClaimStrings
	scopes              map[string][]string
}

func (u *Basic) parseScopes() {
	if u == nil || u.scopes != nil {
		return
	}

	scopes := u.Claim(u.scopeClaimName...)
	if len(scopes) == 1 {
		scopes = strings.Split(scopes[0], u.scopeSeparator)
	}

	u.scopes = make(map[string][]string, len(scopes))
	for _, r := range scopes {
		rr, l, _ := strings.Cut(r, u.scopeLevelSeparator)

		lvls, ok := u.scopes[rr]
		if !ok {
			lvls = []string{l}
		} else {
			lvls = append(lvls, l)
		}

		u.scopes[rr] = lvls
	}
}

// New returns new user instance with specified claims.
func New(claims map[string]token.ClaimStrings, options ...Option) *Basic {
	user := &Basic{
		scopeSeparator:      ",",
		scopeLevelSeparator: ":",
		scopeGroupSeparator: "/",
		scopeClaimName:      []string{"scope", "rights", "http://docs.oasis-open.org/wsfed/authorization/200706/claims/action"},
		claims:              claims,
	}

	for _, opt := range options {
		opt.apply(user)
	}

	user.parseScopes()

	return user
}

// ClaimValue returns claim value.
// If multiple names are provided, claim that matches first name will be returned.
func (u *Basic) ClaimValue(name ...string) string {
	if u == nil {
		return ""
	}

	for _, n := range name {
		if claims, ok := u.claims[n]; ok {
			return claims.Value()
		}
	}

	return ""
}

// Claim returns all claims with specified name.
// If multiple names are provided, claim that matches first name will be returned.
func (u *Basic) Claim(name ...string) token.ClaimStrings {
	if u == nil {
		return nil
	}

	for _, n := range name {
		if claims, ok := u.claims[n]; ok {
			return claims
		}
	}

	return nil
}

// GivenName returns users given name.
func (u *Basic) GivenName() string {
	return u.ClaimValue("given_name", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname")
}

// FamilyName returns users family name.
func (u *Basic) FamilyName() string {
	return u.ClaimValue("family_name", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname")
}

// DisplayName returns users display name.
func (u *Basic) DisplayName() string {
	if name := u.ClaimValue("name", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"); len(name) > 0 {
		return name
	}

	displayName := bytebufferpool.Get()
	defer bytebufferpool.Put(displayName)

	if gn := u.GivenName(); len(gn) > 0 {
		_, _ = displayName.WriteString(gn)
	}

	if fn := u.FamilyName(); len(fn) > 0 {
		if displayName.Len() > 0 {
			_ = displayName.WriteByte(' ')
		}

		_, _ = displayName.WriteString(fn)
	}

	return displayName.String()
}

// Authorized returns if user is authorized.
func (u *Basic) Authorized() bool {
	return u != nil
}

// ID returns user ID.
func (u *Basic) ID() string {
	if u == nil {
		return ""
	}

	return u.ClaimValue("sub", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/nameidentifier")
}

// HasScopeGroup checks if user has any granted scopes in specified group.
func (u *Basic) HasScopeGroup(name string) bool {
	if u == nil {
		return false
	}

	for scope := range u.scopes {
		if scope == name || strings.HasPrefix(scope, name+u.scopeGroupSeparator) {
			return true
		}
	}

	return false
}

// HasScope checks if user has granted scope with any level.
func (u *Basic) HasScope(name string) bool {
	if u == nil {
		return false
	}

	_, ok := u.scopes[name]

	return ok
}

// HasScopeLevel checks if user has granted scope with exact level.
func (u *Basic) HasScopeLevel(name string, level string) bool {
	if u == nil {
		return false
	}

	levels, ok := u.scopes[name]
	if !ok {
		return false
	}

	for _, l := range levels {
		if level == l {
			return true
		}
	}

	return false
}

// HasScopeAnyLevel checks if user has granted scope with one of levels.
func (u *Basic) HasScopeAnyLevel(name string, levels ...string) bool {
	if u == nil {
		return false
	}

	for _, l := range levels {
		if u.HasScopeLevel(name, l) {
			return true
		}
	}

	return false
}
