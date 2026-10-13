package vcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/satoricorp/gx/internal/storage"
)

// RecordGitCommit records metadata for a git commit that already exists.
func (s *Service) RecordGitCommit(ctx context.Context, repo RepoInfo, commitOID string, pending PendingCommitContext) (CommitResult, error) {
	commitOID = strings.TrimSpace(commitOID)
	if commitOID == "" {
		return CommitResult{}, fmt.Errorf("commit oid required")
	}
	message, err := s.runTrimmed(ctx, repo.RootPath, "git", "log", "-1", "--format=%B", commitOID)
	if err != nil {
		return CommitResult{}, fmt.Errorf("read commit message: %w", err)
	}
	if existing, lookupErr := s.commitResultFromDB(ctx, repo, commitOID); lookupErr == nil {
		return existing, nil
	}
	revisionID, err := s.revisionForCommit(ctx, repo, commitOID, message)
	if err != nil {
		return CommitResult{}, err
	}
	filesOut, err := s.runTrimmed(ctx, repo.RootPath, "git", "diff-tree", "--no-commit-id", "--name-only", "-r", commitOID)
	if err != nil {
		return CommitResult{}, fmt.Errorf("list commit files: %w", err)
	}
	parentOID, _ := s.runTrimmed(ctx, repo.RootPath, "git", "rev-parse", commitOID+"^")
	parentOID = strings.TrimSpace(parentOID)
	var parentChangeID *string
	if parentOID != "" {
		if parentRevision := s.recordedRevisionForCommit(ctx, repo, parentOID); parentRevision != "" {
			parentChangeID = &parentRevision
		}
	}
	subject := strings.TrimSpace(strings.Split(message, "\n")[0])
	change := ChangeInfo{
		ChangeID:       revisionID,
		CommitID:       commitOID,
		Description:    subject,
		ParentChangeID: parentChangeID,
		Files:          splitLines(filesOut),
	}
	branch := strings.TrimSpace(pending.Branch)
	if branch == "" {
		branch, _ = s.currentStagedCommitBranch(ctx, repo.RootPath)
	}
	stack, err := s.resolveStagedCommitStack(ctx, repo, branch, parentOID)
	if err != nil {
		return CommitResult{}, err
	}
	// Sessions are deliberately not matched here. A `sessions` row exists
	// because a transcript was observed, never because a commit happened to
	// match one; the links are written at push time from the hunk links the
	// capture pipeline already produces (AttachSessionsFromHunkLinks).
	result := CommitResult{
		Repo:        repo,
		Change:      change,
		Stack:       &stack,
		OperationID: commitOID,
	}
	if err := recordCommit(ctx, result); err != nil {
		return result, err
	}
	return result, nil
}

// RewriteGitCommitOIDs updates stored commit OIDs after history rewrite.
func (s *Service) RewriteGitCommitOIDs(ctx context.Context, repo RepoInfo, oldOID, newOID string) error {
	oldOID = strings.TrimSpace(oldOID)
	newOID = strings.TrimSpace(newOID)
	if oldOID == "" || newOID == "" || oldOID == newOID {
		return nil
	}
	return withRepoIdentityLock(repo.GitCommonDir, func() error {
		store, err := openStore(ctx)
		if err != nil {
			return err
		}
		defer store.Close()
		repoRow, err := store.FindRepoByIdentity(ctx, repo.GitCommonDir, repo.RootPath)
		if err != nil {
			return err
		}
		if repoRow == nil {
			return nil
		}
		change, err := store.FindChangeByCommitID(ctx, repoRow.ID, oldOID)
		if err != nil {
			return err
		}
		if change == nil {
			return s.recoverGitCommitByOID(ctx, store, repo, *repoRow, newOID)
		}
		now := time.Now().UnixMilli()
		if err := store.UpdateChangeCommitID(ctx, change.ID, newOID, now); err != nil {
			return err
		}
		filesOut, _ := s.runTrimmed(ctx, repo.RootPath, "git", "diff-tree", "--no-commit-id", "--name-only", "-r", newOID)
		filesJSON, err := json.Marshal(splitLines(filesOut))
		if err != nil {
			return err
		}
		return store.WriteChangeRevision(ctx, storage.ChangeRevision{
			ChangeID:      change.ID,
			JJCommitID:    newOID,
			JJOperationID: newOID,
			ChangedFiles:  string(filesJSON),
			CreatedAt:     now,
		})
	})
}

func (s *Service) recoverGitCommitByOID(ctx context.Context, store *storage.Store, repo RepoInfo, repoRow storage.Repo, commitOID string) error {
	if existing, err := store.FindChangeByCommitID(ctx, repoRow.ID, commitOID); err != nil || existing != nil {
		return err
	}
	_, err := s.RecordGitCommit(ctx, repo, commitOID, PendingCommitContext{
		WorktreeRoot: repo.RootPath,
		GitCommonDir: repo.GitCommonDir,
	})
	return err
}

// RecoverMissingCommits records the commits ~/.gx has no revision for: ones
// made while gx's hooks were not running. It returns how many it recorded.
func (s *Service) RecoverMissingCommits(ctx context.Context, repo RepoInfo, commitOIDs []string) (int, error) {
	if len(commitOIDs) == 0 {
		return 0, nil
	}
	store, err := openStore(ctx)
	if err != nil {
		return 0, err
	}
	defer store.Close()
	repoRow, err := store.FindRepoByIdentity(ctx, repo.GitCommonDir, repo.RootPath)
	if err != nil {
		return 0, err
	}
	if repoRow == nil {
		return 0, nil
	}
	recovered := 0
	var recoveryErrors []error
	for _, commitOID := range commitOIDs {
		commitOID = strings.TrimSpace(commitOID)
		if commitOID == "" {
			continue
		}
		existing, err := store.FindChangeByCommitID(ctx, repoRow.ID, commitOID)
		if err != nil {
			return recovered, err
		}
		if existing != nil {
			continue
		}
		if _, err := s.RecordGitCommit(ctx, repo, commitOID, PendingCommitContext{
			WorktreeRoot: repo.RootPath,
			GitCommonDir: repo.GitCommonDir,
		}); err != nil {
			recoveryErrors = append(recoveryErrors, fmt.Errorf("record commit %s: %w", shortID(commitOID, 7), err))
			continue
		}
		recovered++
	}
	return recovered, errors.Join(recoveryErrors...)
}

// recordedRevisionForCommit is the revision ~/.gx recorded for a commit, or,
// for a commit made before gx stopped writing trailers, the one its message
// names. It returns "" when neither knows the commit.
func (s *Service) recordedRevisionForCommit(ctx context.Context, repo RepoInfo, commitOID string) string {
	if store, err := openStore(ctx); err == nil {
		defer store.Close()
		if repoRow, err := store.FindRepoByIdentity(ctx, repo.GitCommonDir, repo.RootPath); err == nil && repoRow != nil {
			if change, err := store.FindChangeByCommitID(ctx, repoRow.ID, commitOID); err == nil && change != nil {
				return change.JJChangeID
			}
		}
	}
	message, err := s.runTrimmed(ctx, repo.RootPath, "git", "log", "-1", "--format=%B", commitOID)
	if err != nil {
		return ""
	}
	if ids := ParseRevisionIDsFromMessage(message); len(ids) > 0 {
		return ids[len(ids)-1]
	}
	return ""
}

// revisionForCommit is the revision a newly recorded commit belongs to: the
// one its legacy trailer names, if it has one, otherwise a new one.
func (s *Service) revisionForCommit(ctx context.Context, repo RepoInfo, commitOID, message string) (string, error) {
	if ids := ParseRevisionIDsFromMessage(message); len(ids) > 0 {
		return ids[len(ids)-1], nil
	}
	return GenerateRevisionID()
}
