package provider

import (
	"fmt"

	"github.com/fioncat/grfs/types"
)

func Load(repo *types.Repository, cfg *types.Config) (types.Provider, error) {
	var token string
	if cfg.Auths != nil {
		token = cfg.Auths[repo.Domain]
	}

	var prov types.Provider
	var err error
	if repo.IsGithub() {
		prov = newGithub(repo, token)
	} else {
		prov, err = newGitlab(repo, token)
		if err != nil {
			return nil, fmt.Errorf("init gitlab api: %w", err)
		}
	}

	return prov, nil
}
