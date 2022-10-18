package azugo

import (
	"reflect"

	"github.com/goccy/go-json"
	"github.com/valyala/bytebufferpool"
)

// ClaimStrings is basically just a slice of strings, but it can be either serialized from a string array or just a string.
type ClaimStrings []string

func (s *ClaimStrings) UnmarshalJSON(data []byte) (err error) {
	var value interface{}

	if err = json.Unmarshal(data, &value); err != nil {
		return err
	}

	var aud []string

	switch v := value.(type) {
	case string:
		aud = append(aud, v)
	case []string:
		aud = ClaimStrings(v)
	case []interface{}:
		for _, vv := range v {
			vs, ok := vv.(string)
			if !ok {
				return &json.UnsupportedTypeError{Type: reflect.TypeOf(vv)}
			}
			aud = append(aud, vs)
		}
	case nil:
		return nil
	default:
		return &json.UnsupportedTypeError{Type: reflect.TypeOf(v)}
	}

	*s = aud

	return
}

func (s ClaimStrings) MarshalJSON() (b []byte, err error) {
	if len(s) == 1 {
		return json.Marshal(s[0])
	}

	return json.Marshal([]string(s))
}

// Value of the claim as string.
func (s ClaimStrings) Value() string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

// UserDisplayNamer is an interface that provides method for user display name.
type UserDisplayNamer interface {
	// DisplayName returns user display name.
	DisplayName() string
}

// UserRighter is an interface that provides methods to check user rights.
type UserRighter interface {
	// HasRightGroup checks if user has any rights in specified group.
	HasRightGroup(group string) bool
	// HasRight checks if user has specified right with any right level.
	HasRight(right string) bool
	// HasRightLevel checks if user has specific right level.
	HasRightLevel(right string, level string) bool
}

// UserClaimer is an interface that provides methods to get user claims.
type UserClaimer interface {
	// Claim returns user claim with all values.
	Claim(name ...string) ClaimStrings
	// ClaimValue returns user claim with first value.
	ClaimValue(name ...string) string
}

type User interface {
	UserDisplayNamer
	// UserRighter
	UserClaimer
}

// StandardUser represents authorized user.
type StandardUser struct {
	ID         string                  `json:"id"`
	Authorized bool                    `json:"authorized"`
	Claims     map[string]ClaimStrings `json:"claims"`
	// rights     map[string][]string
}

// NewStandardUser returns new empty user instance
func NewStandardUser() *StandardUser {
	return &StandardUser{
		Claims: make(map[string]ClaimStrings, 10),
	}
}

// ClaimValue returns claim value.
// If multiple names are provided, claim that matches first name will be returned.
func (u *StandardUser) ClaimValue(name ...string) string {
	if u == nil {
		return ""
	}
	for _, n := range name {
		if claims, ok := u.Claims[n]; ok {
			return claims.Value()
		}
	}
	return ""
}

// Claim returns all claims with specified name.
// If multiple names are provided, claim that matches first name will be returned.
func (u *StandardUser) Claim(name ...string) ClaimStrings {
	if u == nil {
		return nil
	}
	for _, n := range name {
		if claims, ok := u.Claims[n]; ok {
			return claims
		}
	}
	return nil
}

// DisplayName returns users display name
func (u *StandardUser) DisplayName() string {
	displayName := bytebufferpool.Get()
	defer bytebufferpool.Put(displayName)

	// TODO: Check alternative claim names and display name claim

	if claim := u.ClaimValue("firstName"); len(claim) > 0 {
		_, _ = displayName.WriteString(claim)
	}
	if claim := u.ClaimValue("lastName"); len(claim) > 0 {
		if displayName.Len() > 0 {
			_ = displayName.WriteByte(' ')
		}
		_, _ = displayName.WriteString(claim)
	}

	return displayName.String()
}

// SetUser sets authorized user.
func (ctx *Context) SetUser(u User) {
	ctx.user = u
}

// User returns authorized user or
func (ctx *Context) User() User {
	return ctx.user
}
