package fs

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/fioncat/grfs/osutils"
	"github.com/fioncat/grfs/types"
	fusefs "github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

var (
	_ = (fusefs.NodeGetattrer)((*testRoot)(nil))
	_ = (fusefs.NodeOnAdder)((*testRoot)(nil))
)

type testRoot struct {
	fusefs.Inode
}

func (r *testRoot) OnAdd(ctx context.Context) {
	ch := r.NewPersistentInode(
		ctx, &fusefs.MemRegularFile{
			Data: []byte("Hello fuse!"),
			Attr: fuse.Attr{
				Mode: 0644,
			},
		}, fusefs.StableAttr{Ino: 2})
	r.AddChild("file.txt", ch, false)
}

func (r *testRoot) Getattr(ctx context.Context, fh fusefs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	return 0
}

func TestMount(t *testing.T) {
	mountPath := "./_test/mnt"

	err := osutils.EnsureDir(mountPath)
	if err != nil {
		t.Fatal(err)
	}

	fs, err := Mount(&testRoot{}, mountPath, &types.Config{
		Fs: &types.FilesystemConfig{
			AllowOthers:  true,
			EntryTimeout: time.Second * 3,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Unmount()

	data, err := os.ReadFile("./_test/mnt/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Hello fuse!" {
		t.Fatalf("Invalid file content %q", string(data))
	}
}
