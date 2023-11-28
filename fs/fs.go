package fs

import (
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/fioncat/grfs/types"
	fusefs "github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
)

type Filesystem struct {
	fuseServer *fuse.Server

	stop     chan struct{}
	stopFlag *uint32
}

func Mount(node fusefs.InodeEmbedder, path string, cfg *types.Config) (*Filesystem, error) {
	rawfs := fusefs.NewNodeFS(node, &fusefs.Options{
		AttrTimeout:     &cfg.Fs.EntryTimeout,
		EntryTimeout:    &cfg.Fs.EntryTimeout,
		NullPermissions: true,
	})

	srv, err := fuse.NewServer(rawfs, path, &fuse.MountOptions{
		AllowOther: cfg.Fs.AllowOthers,
		FsName:     "grfs",
		Name:       "grfs",
		Options:    []string{"ro"},
	})
	if err != nil {
		return nil, fmt.Errorf("Init fuse server: %w", err)
	}

	f := &Filesystem{
		fuseServer: srv,
		stop:       make(chan struct{}),
		stopFlag:   new(uint32),
	}
	f.start()

	return f, nil
}

func (fs *Filesystem) start() {
	go func() {
		logrus.Info("Start fuse.grfs server")
		fs.fuseServer.Serve()
		atomic.StoreUint32(fs.stopFlag, 1)
		close(fs.stop)
	}()

	logrus.Info("Wait fuse.grfs mount")
	fs.fuseServer.WaitMount()
}

func (fs *Filesystem) Unmount() {
	if atomic.LoadUint32(fs.stopFlag) == 1 {
		return
	}

	logrus.Info("Unmount fuse.grfs")
	err := fs.fuseServer.Unmount()
	if err != nil {
		logrus.Errorf("Unmount grfs error: %v", err)
		return
	}
	logrus.Info("Wait fuse server stop")
	fs.fuseServer.Wait()
	atomic.StoreUint32(fs.stopFlag, 1)
}

func (fs *Filesystem) UnmountChan() <-chan struct{} {
	return fs.stop
}

const (
	blockSize         = 4096
	physicalBlockSize = 512
	// physicalBlockRatio is the ratio of blockSize to physicalBlockSize.
	// It can be used to convert from # blockSize-byte blocks to # physicalBlockSize-byte blocks
	physicalBlockRatio = blockSize / physicalBlockSize
)

func getEntryFileMode(ent *types.Entry) uint32 {
	mode := uint32(0644)
	switch {
	case ent.IsDir:
		mode = uint32(os.ModePerm)
		mode |= syscall.S_IFDIR
	case ent.IsSymLink:
		mode |= syscall.S_IFLNK
	default:
		mode |= syscall.S_IFREG
	}

	return mode
}

func getEntryIno(ent *types.Entry) uint64 {
	return uint64(uintptr(unsafe.Pointer(ent)))
}

func defaultStatfs(stat *fuse.StatfsOut) {
	// http://man7.org/linux/man-pages/man2/statfs.2.html
	stat.Blocks = 0 // dummy
	stat.Bfree = 0
	stat.Bavail = 0
	stat.Files = 0 // dummy
	stat.Ffree = 0
	stat.Bsize = blockSize
	stat.NameLen = 1<<32 - 1
	stat.Frsize = blockSize
	stat.Padding = 0
	stat.Spare = [6]uint32{}
}
