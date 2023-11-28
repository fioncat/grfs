package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/fioncat/grfs/osutils"
	"github.com/fioncat/grfs/types"
)

type testEntry struct {
	info *types.Entry

	data []byte

	children []*testEntry
}

type testProvider struct {
	ents []*testEntry
}

func (p *testProvider) ReadDir(ctx context.Context, path string) ([]*types.Entry, error) {
	if path == "/" || path == "" {
		return p.convertEntries(p.ents), nil
	}
	for _, ent := range p.ents {
		if ent.info.Path == path {
			return p.convertEntries(ent.children), nil
		}
		for _, child := range ent.children {
			if child.info.Path == path {
				return p.convertEntries(child.children), nil
			}
		}
	}
	return nil, fmt.Errorf("Could not find path %q", path)
}

func (p *testProvider) ReadFile(ctx context.Context, path string) ([]byte, error) {
	for _, ent := range p.ents {
		if ent.info.Path == path {
			return ent.data, nil
		}
		for _, child := range ent.children {
			if child.info.Path == path {
				return child.data, nil
			}
		}
	}
	return nil, fmt.Errorf("Could not find file %q", path)
}

func (p *testProvider) Check(ctx context.Context) error { return nil }

func (p *testProvider) convertEntries(ents []*testEntry) []*types.Entry {
	result := make([]*types.Entry, len(ents))
	for i, ent := range ents {
		info := ent.info
		if len(ent.data) > 0 {
			info.Size = int64(len(ent.data))
		}
		result[i] = info
	}
	return result
}

func TestNode(t *testing.T) {
	testEntries := []*testEntry{
		{
			info: &types.Entry{
				Path: "dir0",
				Name: "dir0",

				IsDir: true,
			},

			children: []*testEntry{
				{
					info: &types.Entry{
						Path: "dir0/file0.txt",
						Name: "file0.txt",
					},
					data: []byte("Hello, I am from dir0/file0!\nNext line\n\n"),
				},
				{
					info: &types.Entry{
						Path: "dir0/file1.txt",
						Name: "file1.txt",
					},
					data: []byte("Hello, file1, I love coding!\n"),
				},
				{
					info: &types.Entry{
						Path: "dir0/empty_file",
						Name: "empty_file",
					},
					data: []byte(""),
				},
			},
		},

		{
			info: &types.Entry{
				Path: "dir1",
				Name: "dir1",

				IsDir: true,
			},

			children: []*testEntry{
				{
					info: &types.Entry{
						Path:  "dir1/empty_dir",
						Name:  "empty_dir",
						IsDir: true,
					},
				},
				{
					info: &types.Entry{
						Path: "dir1/file",
						Name: "file",
					},
					data: []byte("Hello, grfs!"),
				},
			},
		},
		{
			info: &types.Entry{
				Path: "file.txt",
				Name: "file.txt",
			},
			data: []byte("I am a file from root path"),
		},
		{
			info: &types.Entry{
				Path: "README.md",
				Name: "README.md",
			},
			data: []byte("This is a test filesystem\n\nPlease check sub directory\n"),
		},
	}

	p := &testProvider{ents: testEntries}
	node := NewNode(p)

	mountPath := "_test/node"
	err := osutils.EnsureDir(mountPath)
	if err != nil {
		t.Fatal(err)
	}

	fs, err := Mount(node, mountPath, &types.Config{
		Fs: &types.FilesystemConfig{
			EntryTimeout: time.Second * 3,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Unmount()

	for _, ent := range testEntries {
		path := filepath.Join(mountPath, ent.info.Path)
		if ent.info.IsDir {
			var osEnts []os.DirEntry
			osEnts, err = os.ReadDir(path)
			if err != nil {
				t.Fatalf("Read dir for %q: %v", path, err)
			}
			if len(osEnts) != len(ent.children) {
				t.Fatalf("Unexpect ents length from os: %d, expect %d", len(osEnts), len(ent.children))
			}

			for _, child := range ent.children {
				childPath := filepath.Join(mountPath, child.info.Path)
				if child.info.IsDir {
					osEnts, err = os.ReadDir(childPath)
					if err != nil {
						t.Fatalf("Read dir for child %q: %v", childPath, err)
					}
					if len(osEnts) != 0 {
						t.Fatalf("Expect child dir ents to be zero, found %d", len(osEnts))
					}
					continue
				}

				var data []byte
				data, err = os.ReadFile(childPath)
				if err != nil {
					t.Fatalf("Read child file: %v", err)
				}
				if !reflect.DeepEqual(data, child.data) {
					t.Fatalf("Unexpect child content %q, expect %q", string(data), string(child.data))
				}
			}
			continue
		}

		var data []byte
		data, err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("Read child file: %v", err)
		}
		if !reflect.DeepEqual(data, ent.data) {
			t.Fatalf("Unexpect content %q, expect %q", string(data), string(ent.data))
		}
	}
}
