package cicd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

type GitWatcher struct {
	repositories map[string]*Repository
	webhooks     map[string]*WebhookConfig
	pollInterval time.Duration
	callbacks    []CommitCallback
}

type Repository struct {
	URL         string
	Branch      string
	LastCommit  string
	Credentials *http.BasicAuth
	LocalPath   string
}

type WebhookConfig struct {
	Secret   string
	Events   []string
	Endpoint string
}

type CommitEvent struct {
	RepoURL    string
	Branch     string
	CommitHash string
	Message    string
	Author     string
	Timestamp  time.Time
	Files      []string
}

type CommitCallback func(event CommitEvent) error

func NewGitWatcher(pollInterval time.Duration) *GitWatcher {
	return &GitWatcher{
		repositories: make(map[string]*Repository),
		webhooks:     make(map[string]*WebhookConfig),
		pollInterval: pollInterval,
		callbacks:    make([]CommitCallback, 0),
	}
}

func (gw *GitWatcher) AddRepository(url, branch string, credentials *http.BasicAuth) error {
	key := fmt.Sprintf("%s:%s", url, branch)

	// Clone repository to get initial commit
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:      url,
		Auth:     credentials,
		Progress: nil,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository %s: %w", url, err)
	}

	ref, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	gw.repositories[key] = &Repository{
		URL:         url,
		Branch:      branch,
		LastCommit:  ref.Hash().String(),
		Credentials: credentials,
	}

	log.Printf("Added repository %s (branch: %s) for monitoring", url, branch)
	return nil
}

func (gw *GitWatcher) RemoveRepository(url, branch string) {
	key := fmt.Sprintf("%s:%s", url, branch)
	delete(gw.repositories, key)
	log.Printf("Removed repository %s (branch: %s) from monitoring", url, branch)
}

func (gw *GitWatcher) AddCommitCallback(callback CommitCallback) {
	gw.callbacks = append(gw.callbacks, callback)
}

func (gw *GitWatcher) StartPolling(ctx context.Context) {
	ticker := time.NewTicker(gw.pollInterval)
	defer ticker.Stop()

	log.Printf("Starting Git polling with interval %v", gw.pollInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Git polling")
			return
		case <-ticker.C:
			gw.checkRepositories()
		}
	}
}

func (gw *GitWatcher) checkRepositories() {
	for key, repo := range gw.repositories {
		if err := gw.checkRepository(repo); err != nil {
			log.Printf("Error checking repository %s: %v", key, err)
		}
	}
}

func (gw *GitWatcher) checkRepository(repo *Repository) error {
	// Clone the repository
	gitRepo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:           repo.URL,
		Auth:          repo.Credentials,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", repo.Branch)),
		SingleBranch:  true,
		Progress:      nil,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	ref, err := gitRepo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	currentCommit := ref.Hash().String()

	// Check if there's a new commit
	if currentCommit != repo.LastCommit {
		log.Printf("New commit detected in %s:%s - %s", repo.URL, repo.Branch, currentCommit)

		// Get commit details
		commit, err := gitRepo.CommitObject(ref.Hash())
		if err != nil {
			return fmt.Errorf("failed to get commit object: %w", err)
		}

		// Get changed files
		files, err := gw.getChangedFiles(gitRepo, repo.LastCommit, currentCommit)
		if err != nil {
			log.Printf("Failed to get changed files: %v", err)
			files = []string{} // Continue with empty file list
		}

		// Create commit event
		event := CommitEvent{
			RepoURL:    repo.URL,
			Branch:     repo.Branch,
			CommitHash: currentCommit,
			Message:    commit.Message,
			Author:     commit.Author.Name,
			Timestamp:  commit.Author.When,
			Files:      files,
		}

		// Update last commit
		repo.LastCommit = currentCommit

		// Trigger callbacks
		for _, callback := range gw.callbacks {
			if err := callback(event); err != nil {
				log.Printf("Callback error for commit %s: %v", currentCommit, err)
			}
		}
	}

	return nil
}

func (gw *GitWatcher) getChangedFiles(repo *git.Repository, oldCommit, newCommit string) ([]string, error) {
	if oldCommit == "" {
		return []string{}, nil
	}

	oldHash := plumbing.NewHash(oldCommit)
	newHash := plumbing.NewHash(newCommit)

	oldCommitObj, err := repo.CommitObject(oldHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get old commit: %w", err)
	}

	newCommitObj, err := repo.CommitObject(newHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get new commit: %w", err)
	}

	oldTree, err := oldCommitObj.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get old tree: %w", err)
	}

	newTree, err := newCommitObj.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get new tree: %w", err)
	}

	changes, err := object.DiffTree(oldTree, newTree)
	if err != nil {
		return nil, fmt.Errorf("failed to diff trees: %w", err)
	}

	var files []string
	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			continue // Skip on error
		}
		switch action {
		case 0: // Insert
			files = append(files, change.To.Name)
		case 1: // Delete
			files = append(files, change.From.Name)
		case 2: // Modify
			files = append(files, change.To.Name)
		}
	}

	return files, nil
}

func (gw *GitWatcher) GetRepositories() map[string]*Repository {
	repos := make(map[string]*Repository)
	for k, v := range gw.repositories {
		repos[k] = v
	}
	return repos
}

func (gw *GitWatcher) AddWebhook(repoURL string, config *WebhookConfig) {
	gw.webhooks[repoURL] = config
	log.Printf("Added webhook for repository %s", repoURL)
}

func (gw *GitWatcher) HandleWebhook(repoURL string, payload []byte) error {
	_, exists := gw.webhooks[repoURL]
	if !exists {
		return fmt.Errorf("no webhook configured for repository %s", repoURL)
	}

	// TODO: Implement webhook payload parsing for different Git providers
	// For now, just trigger a repository check
	repo, exists := gw.repositories[repoURL]
	if exists {
		return gw.checkRepository(repo)
	}

	return fmt.Errorf("repository %s not being monitored", repoURL)
}
