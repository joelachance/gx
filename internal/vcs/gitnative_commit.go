package vcs

import (
	"context"
	"fmt"
	"strings"
)

// RunPostCommitHook records the latest git commit as a gx revision.
func (s *Service) RunPostCommitHook(ctx context.Context, repoRoot string) error {
	repo, err := s.ResolveGxRepoAtPath(ctx, repoRoot)
	if err != nil {
		return err
	}
	return withRepoIdentityLock(repo.GitCommonDir, func() error {
		pending, err := readPendingCommitContext(repo.GitDir)
		if err != nil {
			pending = PendingCommitContext{
				WorktreeRoot: repo.RootPath,
				GitCommonDir: repo.GitCommonDir,
			}
		}
		if pending.WorktreeRoot == "" {
			pending.WorktreeRoot = repo.RootPath
		}
		// An amend or a rebase rewrites a commit gx already knows. Git runs
		// post-rewrite right after, mapping the old SHA to the new one, and
		// that carries the existing revision over; recording the new SHA here
		// would split one revision into two.
		if rewrite, _ := s.runTrimmed(ctx, repo.RootPath, "git", "reflog", "-1", "--format=%gs", "HEAD"); isRewriteReflogSubject(rewrite) {
			clearPendingCommitContext(repo.GitDir)
			return nil
		}
		headOID, err := s.runTrimmed(ctx, repo.RootPath, "git", "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		_, recordErr := s.RecordGitCommit(ctx, repo, strings.TrimSpace(headOID), pending)
		clearPendingCommitContext(repo.GitDir)
		return recordErr
	})
}

// RunPostRewriteHook updates rewritten commit OIDs from git post-rewrite stdin.
func (s *Service) RunPostRewriteHook(ctx context.Context, repoRoot string, mappings []CommitOIDMapping) error {
	repo, err := s.ResolveGxRepoAtPath(ctx, repoRoot)
	if err != nil {
		return err
	}
	for _, mapping := range mappings {
		if err := s.RewriteGitCommitOIDs(ctx, repo, mapping.OldOID, mapping.NewOID); err != nil {
			return err
		}
	}
	return nil
}

// CommitOIDMapping pairs old and new commit object IDs from post-rewrite.
type CommitOIDMapping struct {
	OldOID string
	NewOID string
}

// ParsePostRewriteMappings parses git post-rewrite stdin lines.
func ParsePostRewriteMappings(input string) []CommitOIDMapping {
	var out []CommitOIDMapping
	for _, line := range strings.Split(input, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 {
			continue
		}
		out = append(out, CommitOIDMapping{OldOID: fields[0], NewOID: fields[1]})
	}
	return out
}

func (s *Service) commitResultFromDB(ctx context.Context, repo RepoInfo, commitOID string) (CommitResult, error) {
	store, err := openStore(ctx)
	if err != nil {
		return CommitResult{}, err
	}
	defer store.Close()
	repoRow, err := store.FindRepoByIdentity(ctx, repo.GitCommonDir, repo.RootPath)
	if err != nil || repoRow == nil {
		return CommitResult{}, fmt.Errorf("repo not initialized")
	}
	changeRow, err := store.FindChangeByCommitID(ctx, repoRow.ID, commitOID)
	if err != nil || changeRow == nil {
		return CommitResult{}, fmt.Errorf("revision not recorded")
	}
	change := ChangeInfo{
		ChangeID:       changeRow.JJChangeID,
		CommitID:       changeRow.CurrentCommitID,
		Description:    changeRow.Description,
		ParentChangeID: changeRow.ParentChangeID,
	}
	var stack *StackInfo
	if branch, branchErr := s.currentStagedCommitBranch(ctx, repo.RootPath); branchErr == nil {
		if existing, err := store.FindStackByBookmark(ctx, repoRow.ID, branch); err == nil && existing != nil {
			info := stackInfoFromStorage(*existing)
			stack = &info
		}
	}
	return CommitResult{
		Repo:             repo,
		Change:           change,
		Stack:            stack,
		OperationID:      commitOID,
		ProvenanceStatus: "recorded",
	}, nil
}

// isRewriteReflogSubject reports whether HEAD's latest reflog entry is an
// amend or a rebase step, the two rewrites post-rewrite reports. A cherry-pick
// is not one: it makes a new commit, and post-rewrite never hears of it.
func isRewriteReflogSubject(subject string) bool {
	subject = strings.TrimSpace(subject)
	return strings.HasPrefix(subject, "commit (amend)") || strings.HasPrefix(subject, "rebase")
}
