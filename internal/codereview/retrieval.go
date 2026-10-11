package codereview

import (
	"os"
	"strings"
)

// Retrieval runs on the reviewer's own accounts: TurboPuffer stores and
// searches the index, and OpenAI embeds the queries. With neither key set, a
// review reads the change and the files around it, and each retrieval source
// says so in its evidence row.

// rawTurboPufferKeyPresent reports whether the store is reachable at all.
func rawTurboPufferKeyPresent() bool {
	return strings.TrimSpace(os.Getenv("TURBOPUFFER_API_KEY")) != ""
}

// directKeysRemedy is what a user does to turn retrieval on.
const directKeysRemedy = "Set TURBOPUFFER_API_KEY and OPENAI_API_KEY, then run `gx index`, to retrieve from this repository"

// noRetrievalBackend is the status of a source with no TURBOPUFFER_API_KEY to
// retrieve with.
func noRetrievalBackend(source string) EvidenceStatus {
	return EvidenceStatus{
		Source: source,
		State:  EvidenceDisabled,
		Detail: "no TURBOPUFFER_API_KEY",
		Remedy: directKeysRemedy,
	}
}
