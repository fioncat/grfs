package types

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	gitparser "github.com/kubescape/go-git-url"
	githubparserv1 "github.com/kubescape/go-git-url/githubparser/v1"
	gitlabparserv1 "github.com/kubescape/go-git-url/gitlabparser/v1"
	giturl "github.com/whilp/git-urls"
)

type Repository struct {
	Domain string `json:"domain"`

	Owner string `json:"owner"`
	Name  string `json:"name"`

	Ref string `json:"ref"`
}

func (r *Repository) String() string {
	base := fmt.Sprintf("%s:%s/%s", r.Domain, r.Owner, r.Name)
	if r.Ref != "" {
		return fmt.Sprintf("%s@%s", base, r.Ref)
	}
	return base
}

func (r *Repository) IsGithub() bool {
	return githubparserv1.IsHostGitHub(r.Domain)
}

func (r *Repository) Path() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

func (r *Repository) Validate() error {
	if r.Domain == "" {
		return errors.New("invalid repo, domain is empty")
	}
	if r.Owner == "" {
		return errors.New("invalid repo, owner is empty")
	}
	if r.Name == "" {
		return errors.New("invalid repo, name is empty")
	}
	return nil
}

var repoSshUrlRegex = regexp.MustCompile(`^(git@)?([^:]*):([^@]*)(@.*)?$`)

func ParseRepository(url string) (*Repository, error) {
	var ref string
	if !strings.HasPrefix(url, "http") {
		matches := repoSshUrlRegex.FindStringSubmatch(url)
		if len(matches) != 5 {
			return nil, errors.New("invalid ssh clone url, the format is: '[git@]<domain>:<repo-path>[@ref]'")
		}

		ref = strings.TrimSpace(strings.TrimPrefix(matches[4], "@"))
	}

	gitUrl, err := giturl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("parse repo url: %w", err)
	}

	var parsedGitUrl gitparser.IGitURL
	if githubparserv1.IsHostGitHub(gitUrl.Host) {
		parsedGitUrl, err = githubparserv1.NewGitHubParserWithURL(url)
		if err != nil {
			return nil, fmt.Errorf("parse github url: %w", err)
		}
	} else {
		parsedGitUrl, err = gitlabparserv1.NewGitLabParserWithURL(url)
		if err != nil {
			return nil, fmt.Errorf("parse gitlab url: %w", err)
		}
	}

	if ref == "" {
		ref = filepath.Join(parsedGitUrl.GetBranchName(), parsedGitUrl.GetPath())
	}

	repo := &Repository{
		Domain: gitUrl.Hostname(),
		Owner:  parsedGitUrl.GetOwnerName(),
		Name:   parsedGitUrl.GetRepoName(),
		Ref:    ref,
	}
	err = repo.Validate()
	return repo, err
}
