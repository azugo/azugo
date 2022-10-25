package token

import (
	"reflect"

	"github.com/goccy/go-json"
)

// ClaimStrings is basically just a slice of strings, but it can be either serialized from a string array or just a string.
type ClaimStrings []string

func (s *ClaimStrings) UnmarshalJSON(data []byte) (err error) {
	var value interface{}

	if err = json.Unmarshal(data, &value); err != nil {
		return err
	}

	var val []string

	switch v := value.(type) {
	case string:
		val = append(val, v)
	case []string:
		val = ClaimStrings(v)
	case []interface{}:
		for _, vv := range v {
			vs, ok := vv.(string)
			if !ok {
				return &json.UnsupportedTypeError{Type: reflect.TypeOf(vv)}
			}
			val = append(val, vs)
		}
	case nil:
		return nil
	default:
		return &json.UnsupportedTypeError{Type: reflect.TypeOf(v)}
	}

	*s = val

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
