package main

import (
	"testing"
)

// TestMain validates that the main package compiles and can be imported.
// The main() function is difficult to test directly due to its use of
// os.Exit and infinite server loop, so we only verify compilation here.
func TestMain(t *testing.T) {
	// This test simply validates that the package compiles
	// and can be imported without errors.
	t.Log("main package compiles successfully")
}
