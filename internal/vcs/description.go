package vcs

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	PlaceholderDescription      = "(no description set)"
	PendingRemainderDescription = "gx: pending remainder"
	// RevisionTrailerFormat is the gx identity trailer line template.
	//
	// This used to be a URL: "gx: https://gx.run/r/<id>". The page it named
	// went away with the rest of the site, which left every commit gx had ever
	// stamped ending in a dead link.
	//
	// The trailer itself cannot go, because it is the index. Resolving a
	// revision ID to a commit is `git log --grep=<trailer>`
	// (gitChangeByRevisionID), recovery is the same search (findCommitByRevisionID),
	// and "which revisions are in this range" parses the ids back out of the
	// messages (ParseRevisionIDsFromMessage). Remove the line and a revision
	// stops being findable at all, which is a worse break than a stale link.
	// So the marker stays and the URL goes: it carries the same information and
	// promises nothing.
	RevisionTrailerFormat = "gx-revision: %s"

	// LegacyRevisionTrailerPrefix is the URL form this used to stamp. It is
	// still parsed, so history written before this change keeps resolving:
	// without it every commit already in a repository would stop being
	// recognised as carrying a gx revision the day the format changed.
	LegacyRevisionTrailerPrefix = "gx: https://gx.run/r/"
)

var (
	ErrNoRecordedAdds     = errors.New("no commit recorded for this stack")
	ErrEmptyCommitMessage = errors.New("commit message is required")
)

func IsPlaceholderDescription(desc string) bool {
	trimmed := strings.TrimSpace(desc)
	if trimmed == "" {
		return true
	}
	return strings.EqualFold(trimmed, PlaceholderDescription)
}

func ValidateCommitMessage(message string) error {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return fmt.Errorf("%w; use -m \"describe this revision\"", ErrEmptyCommitMessage)
	}
	if IsPlaceholderDescription(trimmed) {
		return fmt.Errorf("commit message cannot be %q", PlaceholderDescription)
	}
	return nil
}

func RevisionTrailerLine(revisionID string) string {
	revisionID = strings.TrimSpace(revisionID)
	if revisionID == "" {
		return ""
	}
	return fmt.Sprintf(RevisionTrailerFormat, revisionID)
}

// StampRevisionTrailer returns message carrying exactly one gx identity
// trailer for revisionID. gx trailers from a prior revision identity are
// replaced, never accumulated.
func StampRevisionTrailer(message, revisionID string) string {
	revisionID = strings.TrimSpace(revisionID)
	if revisionID == "" {
		return message
	}
	trailer := RevisionTrailerLine(revisionID)
	lines := strings.Split(message, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if isRevisionTrailerLine(line) {
			continue
		}
		kept = append(kept, line)
	}
	body := strings.TrimRight(strings.Join(kept, "\n"), "\n")
	if body == "" {
		return trailer
	}
	return body + "\n\n" + trailer
}

func isRevisionTrailerLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), revisionTrailerPrefix())
}

// revisionTrailerPresent reports whether message already carries the canonical
// gx trailer for revisionID and no stale trailers, i.e. re-stamping would be a
// no-op.
func revisionTrailerPresent(message, revisionID string) bool {
	return StampRevisionTrailer(message, revisionID) == strings.TrimRight(message, "\n")
}

func revisionTrailerPrefix() string {
	return fmt.Sprintf(RevisionTrailerFormat, "")
}

func FinalizeCommitMessage(message, revisionID string) (string, error) {
	if err := ValidateCommitMessage(message); err != nil {
		return "", err
	}
	return StampRevisionTrailer(message, revisionID), nil
}

func validateRecordedChangeDescription(description string) error {
	if err := ValidateCommitMessage(description); err != nil {
		return fmt.Errorf("recorded revision has invalid description: %w", err)
	}
	return nil
}

func (s *Service) requireRecordedAddsForPublish(ctx context.Context, stack StackInfo) error {
	if stack.ID == 0 {
		return fmt.Errorf(
			"%w: stage changes with `git add`, then run `git commit -m \"describe this revision\"` before `git push`",
			ErrNoRecordedAdds,
		)
	}
	store, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	changes, err := store.ListChangesByStackID(ctx, stack.ID)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		return fmt.Errorf(
			"%w: stage changes with `git add`, then run `git commit -m \"...\"` to record at least one change before `git push`",
			ErrNoRecordedAdds,
		)
	}
	for _, change := range changes {
		if !IsPlaceholderDescription(change.Description) {
			return nil
		}
	}
	return fmt.Errorf(
		"%w: recorded changes have no description; stage changes with `git add`, then run `git commit -m \"...\"`",
		ErrNoRecordedAdds,
	)
}
