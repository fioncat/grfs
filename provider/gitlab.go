package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/fioncat/grfs/types"
	"github.com/xanzy/go-gitlab"
)

type gitlabProvider struct {
	repo *types.Repository

	client *gitlab.Client
}

func newGitlab(repo *types.Repository, token string) (types.Provider, error) {
	url := fmt.Sprintf("https://%s/api/v4", repo.Domain)
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(url))
	if err != nil {
		return nil, err
	}

	return &gitlabProvider{
		repo:   repo,
		client: client,
	}, nil
}

func (p *gitlabProvider) Check(ctx context.Context) error {
	project, _, err := p.client.Projects.GetProject(p.repo.Path(), &gitlab.GetProjectOptions{})
	if err != nil {
		return fmt.Errorf("gitlab get project: %w", err)
	}
	if p.repo.Ref == "" {
		p.repo.Ref = project.DefaultBranch
	}

	return nil
}

func (p *gitlabProvider) ReadDir(ctx context.Context, path string) ([]*types.Entry, error) {
	nodes, _, err := p.client.Repositories.ListTree(p.repo.Path(), &gitlab.ListTreeOptions{
		Path: gitlab.Ptr(path),
		Ref:  &p.repo.Ref,
	})
	if err != nil {
		fmt.Printf("Path: %q\n", path)
		return nil, err
	}

	ents := make([]*types.Entry, len(nodes))
	for i, node := range nodes {
		if node.Path == "" || node.Name == "" {
			return nil, errors.New("Gitlab return entry with empty name or path")
		}

		var isDir bool
		var size int64
		switch node.Type {
		case "tree":
			isDir = true

		case "blob":
			// TODO: Handle symlink, use node.Mode
			// FIXME: How to get the web url for gitlab entry?

			fileMeta, _, err := p.client.RepositoryFiles.GetFileMetaData(p.repo.Path(), node.Path, &gitlab.GetFileMetaDataOptions{
				Ref: &p.repo.Ref,
			})
			if err != nil {
				return nil, fmt.Errorf("Get file meta for %q: %w", node.Path, err)
			}
			size = int64(fileMeta.Size)

		}

		ents[i] = &types.Entry{
			Path:  node.Path,
			Name:  node.Name,
			IsDir: isDir,
			Size:  size,
		}
	}

	return ents, nil
}

func (p *gitlabProvider) ReadFile(ctx context.Context, path string) ([]byte, error) {
	data, _, err := p.client.RepositoryFiles.GetRawFile(p.repo.Path(), path, &gitlab.GetRawFileOptions{
		Ref: &p.repo.Ref,
	})
	return data, err
}
