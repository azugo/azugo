package azugo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImplementsContextInterface(t *testing.T) {
	assert.Implements(t, (*context.Context)(nil), &Context{})
}
