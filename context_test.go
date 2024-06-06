package azugo

import (
	"context"
	"testing"

	"github.com/go-quicktest/qt"
)

func TestImplementsContextInterface(t *testing.T) {
	qt.Check(t, qt.Implements[context.Context](&Context{}))
}
