package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/fioncat/grfs/types"
	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

type githubProvider struct {
	repo *types.Repository

	client *github.Client
}

func newGithub(repo *types.Repository, token string) types.Provider {
	var httpCli *http.Client
	ctx := context.Background()
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
		})
		httpCli = oauth2.NewClient(ctx, ts)
	}

	client := github.NewClient(httpCli)

	return &githubProvider{
		repo:   repo,
		client: client,
	}
}

func (p *githubProvider) Check(ctx context.Context) error {
	githubRepo, _, err := p.client.Repositories.Get(ctx, p.repo.Owner, p.repo.Name)
	if err != nil {
		return fmt.Errorf("github get repository: %w", err)
	}
	if p.repo.Ref == "" {
		if githubRepo.DefaultBranch != nil {
			p.repo.Ref = *githubRepo.DefaultBranch
		}
	}
	return nil
}

func (p *githubProvider) ReadDir(ctx context.Context, path string) ([]*types.Entry, error) {
	fc, dc, _, err := p.client.Repositories.GetContents(ctx, p.repo.Owner, p.repo.Name, path,
		&github.RepositoryContentGetOptions{
			Ref: p.repo.Ref,
		})
	if err != nil {
		return nil, err
	}
	if fc != nil {
		return nil, fmt.Errorf("%q is a file, not directory", path)
	}

	ents := make([]*types.Entry, len(dc))
	for i, content := range dc {
		path := content.GetPath()
		name := content.GetName()
		if path == "" || name == "" {
			return nil, errors.New("github return entry with empty name or path")
		}

		var (
			isDir     bool
			isSymLink bool
			linkName  string

			size int64
		)
		switch content.GetType() {
		case "dir":
			isDir = true

		case "file":
			size = int64(content.GetSize())

		case "symlink":
			isSymLink = true
			linkName = content.GetTarget()
			if linkName == "" {
				return nil, fmt.Errorf("entry %q is a symlink, but its target is empty", path)
			}

		case "":
			return nil, fmt.Errorf("entry type is empty for %q", path)

		default:
			return nil, fmt.Errorf("unknown entry type %q for %q", content.GetType(), path)
		}

		ents[i] = &types.Entry{
			Path:      path,
			Name:      name,
			IsDir:     isDir,
			IsSymLink: isSymLink,
			LinkName:  linkName,
			Size:      size,
			WebUrl:    content.GetHTMLURL(),
		}
	}

	return ents, nil
}

func (p *githubProvider) ReadFile(ctx context.Context, path string) ([]byte, error) {
	reader, _, err := p.client.Repositories.DownloadContents(ctx, p.repo.Owner, p.repo.Name, path,
		&github.RepositoryContentGetOptions{
			Ref: p.repo.Ref,
		})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read content for %q: %w", path, err)
	}

	return data, nil
}
