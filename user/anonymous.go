package user

import (
	"azugo.io/azugo/token"
)

// Anonymous is a user that is not authorized.
type Anonymous struct{}

// ClaimValue returns claim value.
// If multiple names are provided, claim that matches first name will be returned.
func (u Anonymous) ClaimValue(name ...string) string {
	return ""
}

// Claim returns all claims with specified name.
// If multiple names are provided, claim that matches first name will be returned.
func (u Anonymous) Claim(name ...string) token.ClaimStrings {
	return nil
}

// GivenName returns users given name.
func (u Anonymous) GivenName() string {
	return ""
}

// FamilyName returns users family name.
func (u Anonymous) FamilyName() string {
	return ""
}

// DisplayName returns users display name.
func (u Anonymous) DisplayName() string {
	return ""
}

// Authorized returns if user is authorized.
func (u Anonymous) Authorized() bool {
	return false
}

// HasScopeGroup checks if user has any granted scopes in specified group.
func (u Anonymous) HasScopeGroup(name string) bool {
	return false
}

// HasScope checks if user has granted scope with any level.
func (u Anonymous) HasScope(name string) bool {
	return false
}

// HasScopeLevel checks if user has granted scope with exact level.
func (u Anonymous) HasScopeLevel(name string, level string) bool {
	return false
}

// HasScopeLevel checks if user has granted scope with exact level.
func (u Anonymous) HasScopeAnyLevel(name string, levels []string) bool {
	return false
}

// ID returns user ID.
func (u Anonymous) ID() string {
	return ""
}
