package integration

import (
	"os"
	"testing"

	"portway/internal/codexbridge/fakecli"
)

// TestMain lets this test binary act as the fake Codex/Claude CLI for bridge fixtures.
func TestMain(m *testing.M) {
	if fakecli.Active() {
		fakecli.Main()
	}
	os.Exit(m.Run())
}
