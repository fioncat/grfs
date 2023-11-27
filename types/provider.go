package types

import "context"

type Entry struct {
	Path string
	Name string

	IsDir bool

	IsSymLink bool
	LinkName  string

	Size int64

	WebUrl string
}

type Provider interface {
	ReadDir(ctx context.Context, path string) ([]*Entry, error)
	ReadFile(ctx context.Context, path string) ([]byte, error)
}
