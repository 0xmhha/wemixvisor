package config

import (
	"os"
	"testing"
)

var portCheckOriginal func(int) bool

func TestMain(m *testing.M) {
	portCheckOriginal = portAvailabilityCheck
	portAvailabilityCheck = func(int) bool { return true }
	code := m.Run()
	portAvailabilityCheck = portCheckOriginal
	os.Exit(code)
}
