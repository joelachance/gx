package github

import (
	"os"
	"testing"
)

// TestMain keeps the whole package away from the machine's real GitHub CLI
// login. Its token is a live credential: a test that reached it by accident
// would print it the first time an assertion about the token failed. Tests that
// exercise the fallback set their own stub with stubGHCLI.
func TestMain(m *testing.M) {
	ghCLIToken = func() string { return "" }
	os.Exit(m.Run())
}
