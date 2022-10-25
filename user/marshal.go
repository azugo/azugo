package user

import (
	"azugo.io/azugo/token"

	"github.com/goccy/go-json"
)

type basicClaims struct {
	Claims map[string]token.ClaimStrings `json:"claims"`
}

func (u *Basic) UnmarshalJSON(data []byte) error {
	var value basicClaims

	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	u.claims = value.Claims
	u.parseScopes()

	return nil
}

func (u Basic) MarshalJSON() (b []byte, err error) {
	return json.Marshal(basicClaims{
		Claims: u.claims,
	})
}
