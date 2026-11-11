package utils

import (
	"testing"

	"github.com/go-quicktest/qt"
)

func TestParseBoolValue(t *testing.T) {
	for _, v := range []string{"true", "TRUE", "True", "1", "on", "ON", "On"} {
		qt.Check(t, qt.IsTrue(ParseBoolValue(v)), qt.Commentf("%q", v))
	}

	for _, v := range []string{"", "false", "0", "off", "yes", "no", "2", " true"} {
		qt.Check(t, qt.IsFalse(ParseBoolValue(v)), qt.Commentf("%q", v))
	}
}
