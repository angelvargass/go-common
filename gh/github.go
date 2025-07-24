package gh

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v73/github"
)

// New creates a new instance of the Github client with the provided logger and authentication token.
func New(logger *slog.Logger, token string) *Github {
	client := github.NewClient(nil).WithAuthToken(token)
	return &Github{
		Logger: logger.With("common", "github"),
		Client: client,
	}
}

// GetRepository gets a repository as specified by the owner/name parameters.
func (gh *Github) GetRepository(ctx context.Context, owner, name string) (*github.Repository, error) {
	gh.Logger.Debug("get repository", slog.String("owner", owner), slog.String("repo name", name))
	repo, res, err := gh.Client.Repositories.Get(ctx, owner, name)
	if res.StatusCode == 404 {
		gh.Logger.Debug("repository not found", slog.String("owner", owner), slog.String("repo name", name))
	}

	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("error getting repository %s/%s", owner, name), err)
	}

	return repo, nil
}

// CreateRepository creates a new repository as specified by the organization/name.
//
// If authenticated as user, pass an empty organization string to create the repository under the authenticated user.
// Repositories created by this function are public by default (private repositories are not created due to rulesets limitations).
//
// Default branch name is set to your configuration on Github.
// Branches are deleted when merged by default.
// A README.md file is created by default.
//
// Changes can have propagation time on GH's servers.
func (gh *Github) CreateRepository(ctx context.Context, organization, name string) (*github.Repository, error) {
	gh.Logger.Debug("creating repository", slog.String("organization", organization), slog.String("name", name))
	repo, res, err := gh.Client.Repositories.Create(ctx, "", &github.Repository{
		Name:                      github.Ptr(name),
		Private:                   github.Ptr(false),
		HasIssues:                 github.Ptr(true),
		HasProjects:               github.Ptr(false),
		HasWiki:                   github.Ptr(false),
		AutoInit:                  github.Ptr(true),
		HasDiscussions:            github.Ptr(true),
		DeleteBranchOnMerge:       github.Ptr(true),
		UseSquashPRTitleAsDefault: github.Ptr(true),
		AllowForking:              github.Ptr(true),
	})

	if res.StatusCode == 422 {
		gh.Logger.Debug("validation failed", slog.String("organization", organization), slog.String("name", name))
	}

	if err != nil {
		gh.Logger.Debug("error creating repository", slog.String("organization", organization), slog.String("name", name))
		return nil, err
	}

	return repo, nil
}

// CreateBranch creates a new branch in the specified repository.
//
// Takes the last commit from the default branch and creates a new branch with the specified name.
func (gh *Github) CreateBranch(ctx context.Context, owner, repoName, branchName string) (*github.Reference, error) {
	gh.Logger.Debug("creating branch", slog.String("owner", owner), slog.String("repo name", repoName), slog.String("branch name", branchName))
	repo, err := gh.GetRepository(ctx, owner, repoName)
	if err != nil {
		gh.Logger.Debug("error getting repository", slog.String("owner", owner), slog.String("repo name", repoName))
		return nil, err
	}

	gh.Logger.Debug("getting latest reference from default branch", slog.String("owner", owner), slog.String("repo name", repoName), slog.String("default branch name", *repo.DefaultBranch))
	ref, _, err := gh.Client.Git.GetRef(ctx, owner, repoName, "refs/heads/"+*repo.DefaultBranch)
	if err != nil {
		gh.Logger.Debug("error getting latest reference from default branch", slog.String("owner", owner), slog.String("repo name", repoName), slog.String("default branch name", *repo.DefaultBranch))
		return nil, err
	}

	branch, _, err := gh.Client.Git.CreateRef(ctx, owner, repoName, &github.Reference{
		Ref: github.Ptr("refs/heads/" + branchName),
		Object: &github.GitObject{
			SHA: ref.Object.SHA,
		},
	})

	if err != nil {
		gh.Logger.Debug("error creating new branch", slog.String("owner", owner), slog.String("repo name", repoName), slog.String("new branch name", branchName))
		return nil, err
	}

	return branch, nil
}

// CreateOrUpdateFile creates or updates a file in the specified repository and branch.
//
// If a file is being updated, a SHA is required for the file that is being updated.
// Returns the parsed response from CreateFile operation in the Github's API.
func (gh *Github) CreateOrUpdateFile(ctx context.Context, owner, repoName, branch, commitMessage, filePath, replacingFileSHA string, fileContent []byte) (*github.RepositoryContentResponse, error) {
	gh.Logger.Debug("creating file", slog.String("repo name", repoName), slog.String("branch name", branch), slog.String("file path", filePath))
	content, _, err := gh.Client.Repositories.CreateFile(ctx, owner, repoName, filePath, &github.RepositoryContentFileOptions{
		Message: github.Ptr(commitMessage),
		Content: fileContent,
		SHA:     github.Ptr(replacingFileSHA),
		Branch:  github.Ptr(branch),
	})

	if err != nil {
		gh.Logger.Debug("error creating file", slog.String("repo name", repoName), slog.String("branch name", branch), slog.String("file path", filePath))
		return nil, err
	}

	return content, nil
}

// GetRepositoryContent gets the content of a repository.
//
// A path can be specified. If an empty path is passed, the function will return the content of the root directory.
// A ref can be specified.
func (gh *Github) GetRepositoryContent(ctx context.Context, owner, repoName, path, ref string) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, err error) {
	gh.Logger.Debug("getting repository content", slog.String("repo name", repoName), slog.String("ref", ref), slog.String("path", path))
	fileContents, dirContents, res, err := gh.Client.Repositories.GetContents(ctx, owner, repoName, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})

	if res.StatusCode == 404 {
		gh.Logger.Debug("file/path not found", slog.String("repo name", repoName), slog.String("ref", ref), slog.String("path", path))
	}

	if err != nil && res.StatusCode != 404 {
		gh.Logger.Debug("error getting repository content", slog.String("repo name", repoName), slog.String("ref", ref), slog.String("path", path))
		return nil, nil, err
	}

	return fileContents, dirContents, nil
}

// ReplaceTopics replaces the topics of a repository.
func (gh *Github) ReplaceTopics(ctx context.Context, owner, repoName string, topics []string) ([]string, error) {
	gh.Logger.Debug("replacing repository topics", slog.String("repo name", repoName), slog.Any("topics", topics))
	topics, _, err := gh.Client.Repositories.ReplaceAllTopics(ctx, owner, repoName, topics)

	if err != nil {
		gh.Logger.Debug("error replacing repository topics", slog.String("repo name", repoName), slog.Any("topics", topics))
		return nil, err
	}

	return topics, nil
}

// CreateOrUpdateRepositorySecret creates or updates a secret in the specified repository.
//
// The secret value is encrypted using the repository's public key.
func (gh *Github) CreateOrUpdateRepositorySecret(ctx context.Context, owner, repoName, secretName, secretValue string) error {
	gh.Logger.Debug("creating or updating repository secret", slog.String("repo name", repoName), slog.String("secret name", secretName))
	key, _, err := gh.Client.Actions.GetRepoPublicKey(ctx, owner, repoName)
	if err != nil {
		gh.Logger.Debug("error getting repository public key", slog.String("repo name", repoName))
	}

	encryptedSecret, err := gh.encryptSecret(*key.Key, secretValue)
	if err != nil {
		gh.Logger.Debug("error encrypting secret", slog.String("repo name", repoName), slog.String("secret name", secretName))
		return err
	}

	_, err = gh.Client.Actions.CreateOrUpdateRepoSecret(ctx, owner, repoName, &github.EncryptedSecret{
		Name:           secretName,
		KeyID:          *key.KeyID,
		EncryptedValue: encryptedSecret,
	})

	if err != nil {
		gh.Logger.Debug("error creating or updating repo secret", slog.String("repo name", repoName), slog.String("secret name", secretName))
		return err
	}

	return nil
}

// CreatePullRequest creates a pull request in the specified repository.
//
// The pull request is created from the specified head branch to the base branch.
func (gh *Github) CreatePullRequest(ctx context.Context, owner, repoName, title, body, head, base string) (*github.PullRequest, error) {
	gh.Logger.Debug("creating pull request", slog.String("repo name", repoName), slog.String("title", title), slog.String("head", head), slog.String("base", base))
	pr, _, err := gh.Client.PullRequests.Create(ctx, owner, repoName, &github.NewPullRequest{
		Title: github.Ptr(title),
		Body:  github.Ptr(body),
		Head:  github.Ptr(head),
		Base:  github.Ptr(base),
	})

	if err != nil {
		gh.Logger.Debug("error creating pull request", slog.String("repo name", repoName), slog.String("title", title), slog.String("head", head), slog.String("base", base))
		return nil, err
	}

	return pr, nil
}

// CreateRepositoryRuleset creates a repository ruleset for the specified repository.
//
// The provided rule is applied to the default branch of the repository.
func (gh *Github) CreateRepositoryRuleset(ctx context.Context, owner, repoName, rulesetName string, rules *github.RepositoryRulesetRules) (*github.RepositoryRuleset, error) {
	gh.Logger.Debug("creating repository ruleset", slog.String("repo name", repoName))

	ruleset, _, err := gh.Client.Repositories.CreateRuleset(ctx, owner, repoName, github.RepositoryRuleset{
		Name:        rulesetName,
		Target:      github.Ptr(github.RulesetTargetBranch),
		Enforcement: github.RulesetEnforcementActive,
		Conditions: &github.RepositoryRulesetConditions{
			RefName: &github.RepositoryRulesetRefConditionParameters{
				Include: []string{"~DEFAULT_BRANCH"},
				Exclude: []string{},
			},
		},
		Rules: rules,
	})

	if err != nil {
		gh.Logger.Debug("error creating repository ruleset", slog.String("repo name", repoName))
		return nil, err
	}

	return ruleset, nil
}
