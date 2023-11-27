package types

import (
	"reflect"
	"testing"
)

func TestParseRepository(t *testing.T) {
	testCases := []struct {
		url    string
		expect *Repository
	}{
		{
			url: "https://my-gitlab.com/test-group/test-repo.git",
			expect: &Repository{
				Domain: "my-gitlab.com",

				Owner: "test-group",
				Name:  "test-repo",

				Ref: "",
			},
		},
		{
			url: "https://my-gitlab.com/test-group/test-repo",
			expect: &Repository{
				Domain: "my-gitlab.com",

				Owner: "test-group",
				Name:  "test-repo",

				Ref: "",
			},
		},
		{
			url: "https://my-gitlab.com/k8s/devops/etcdhelper/-/tree/feat/errlog",
			expect: &Repository{
				Domain: "my-gitlab.com",

				Owner: "k8s/devops",
				Name:  "etcdhelper",

				Ref: "feat/errlog",
			},
		},
		{
			url: "http://my-gitlab.com:1278/test-group/test-repo",
			expect: &Repository{
				Domain: "my-gitlab.com",

				Owner: "test-group",
				Name:  "test-repo",

				Ref: "",
			},
		},
		{
			url: "http://my-gitlab.com:3345/k8s/devops/etcdhelper/-/tree/feat/errlog",
			expect: &Repository{
				Domain: "my-gitlab.com",

				Owner: "k8s/devops",
				Name:  "etcdhelper",

				Ref: "feat/errlog",
			},
		},
		{
			url: "gitlab.com:k8s/devops/mocker@feat/data-mysql",
			expect: &Repository{
				Domain: "gitlab.com",

				Owner: "k8s/devops",
				Name:  "mocker",

				Ref: "feat/data-mysql",
			},
		},
		{
			url: "git@gitlab.com:k8s-test/sender@fix/errormsg",
			expect: &Repository{
				Domain: "gitlab.com",

				Owner: "k8s-test",
				Name:  "sender",

				Ref: "fix/errormsg",
			},
		},
		{
			url: "https://github.com/golang/go/tree/release-branch.go1.21",
			expect: &Repository{
				Domain: "github.com",

				Owner: "golang",
				Name:  "go",

				Ref: "release-branch.go1.21",
			},
		},
		{
			url: "git@github.com:golang/go.git",
			expect: &Repository{
				Domain: "github.com",

				Owner: "golang",
				Name:  "go",

				Ref: "",
			},
		},
		{
			url: "github.com:fioncat/grfs.git",
			expect: &Repository{
				Domain: "github.com",

				Owner: "fioncat",
				Name:  "grfs",

				Ref: "",
			},
		},
		{
			url: "github.com:fioncat/grfs",
			expect: &Repository{
				Domain: "github.com",

				Owner: "fioncat",
				Name:  "grfs",

				Ref: "",
			},
		},
		{
			url: "git@github.com:fioncat/grfs",
			expect: &Repository{
				Domain: "github.com",

				Owner: "fioncat",
				Name:  "grfs",

				Ref: "",
			},
		},
		{
			url: "git@github.com:fioncat/grfs.git",
			expect: &Repository{
				Domain: "github.com",

				Owner: "fioncat",
				Name:  "grfs",

				Ref: "",
			},
		},
		{
			url: "git@github.com:fioncat/grfs.git@dev",
			expect: &Repository{
				Domain: "github.com",

				Owner: "fioncat",
				Name:  "grfs",

				Ref: "dev",
			},
		},
		{
			url: "git@github.com:fioncat/grfs.git@feat/database",
			expect: &Repository{
				Domain: "github.com",

				Owner: "fioncat",
				Name:  "grfs",

				Ref: "feat/database",
			},
		},
		{
			url: "https://github.com/kubernetes/kubernetes/tree/feature-serverside-apply",
			expect: &Repository{
				Domain: "github.com",

				Owner: "kubernetes",
				Name:  "kubernetes",

				Ref: "feature-serverside-apply",
			},
		},
	}

	for i, tc := range testCases {
		repo, err := ParseRepository(tc.url)
		if err != nil {
			t.Fatalf("Parse url %q: %v", tc.url, err)
		}
		if !reflect.DeepEqual(repo, tc.expect) {
			t.Fatalf("Unexpect parsed repo %+v, expect %+v, Index: %d", repo, tc.expect, i)
		}
	}
}

func TestRepositoryString(t *testing.T) {
	testCases := []struct {
		repo     *Repository
		str      string
		isGithub bool
	}{
		{
			repo: &Repository{
				Domain: "github.com",
				Owner:  "fioncat",
				Name:   "grfs",
			},
			str:      "github.com:fioncat/grfs",
			isGithub: true,
		},
		{
			repo: &Repository{
				Domain: "github.com",
				Owner:  "fioncat",
				Name:   "grfs",
				Ref:    "dev",
			},
			str:      "github.com:fioncat/grfs@dev",
			isGithub: true,
		},
		{
			repo: &Repository{
				Domain: "my-gitlab.com",
				Owner:  "k8s/devops",
				Name:   "etcdhelper",
				Ref:    "",
			},
			str:      "my-gitlab.com:k8s/devops/etcdhelper",
			isGithub: false,
		},
		{
			repo: &Repository{
				Domain: "my-gitlab.com",
				Owner:  "k8s/devops",
				Name:   "etcdhelper",
				Ref:    "feat/errlog",
			},
			str:      "my-gitlab.com:k8s/devops/etcdhelper@feat/errlog",
			isGithub: false,
		},
	}

	for i, tc := range testCases {
		str := tc.repo.String()
		isGithub := tc.repo.IsGithub()

		if str != tc.str {
			t.Fatalf("Unexpect repo string %q, expect %q, index %d", str, tc.str, i)
		}
		if isGithub != tc.isGithub {
			t.Fatalf("Unexpect repo isGithub %v, expect %v, index %d", isGithub, tc.isGithub, i)
		}
	}
}
