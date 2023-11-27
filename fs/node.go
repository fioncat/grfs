package fs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/user"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fioncat/grfs/types"
	fusefs "github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/sirupsen/logrus"
)

var (
	_ = (fusefs.InodeEmbedder)((*Node)(nil))

	_ = (fusefs.NodeReaddirer)((*Node)(nil))
	_ = (fusefs.NodeLookuper)((*Node)(nil))

	_ = (fusefs.NodeGetattrer)((*Node)(nil))
	_ = (fusefs.NodeReadlinker)((*Node)(nil))
	_ = (fusefs.NodeStatfser)((*Node)(nil))

	_ = (fusefs.NodeGetxattrer)((*Node)(nil))
	_ = (fusefs.NodeListxattrer)((*Node)(nil))

	_ = (fusefs.NodeOpener)((*Node)(nil))
	_ = (fusefs.FileReader)((*Node)(nil))
)

type Node struct {
	fusefs.Inode

	provider types.Provider

	entry *types.Entry

	logger *logrus.Entry

	createTime time.Time

	subDirEnts []fuse.DirEntry
	subEnts    []*types.Entry
	subCache   bool
	subMu      sync.Mutex

	reader        *bytes.Reader
	readContentMu sync.Mutex
}

func NewNode(ent *types.Entry, provider types.Provider) *Node {
	logger := logrus.WithFields(logrus.Fields{
		"Path":      ent.Path,
		"IsDir":     ent.IsDir,
		"IsSymLink": ent.IsSymLink,
		"LinkName":  ent.LinkName,
		"Size":      ent.Size,
	})

	return &Node{
		provider: provider,
		entry:    ent,

		createTime: time.Now(),

		logger: logger,
	}
}

func (n *Node) Readdir(ctx context.Context) (fusefs.DirStream, syscall.Errno) {
	ents, err := n.listSubEntries(ctx)
	if err != nil {
		n.logger.Errorf("List sub entries error: %v", err)
		return nil, syscall.EIO
	}

	return fusefs.NewListDirStream(ents), 0
}

func (n *Node) listSubEntries(ctx context.Context) ([]fuse.DirEntry, error) {
	n.subMu.Lock()
	if n.subCache {
		ents := n.subDirEnts
		n.subMu.Unlock()
		return ents, nil
	}
	n.subMu.Unlock()

	start := time.Now()
	ents, err := n.provider.ReadDir(context.Background(), n.entry.Path)
	if err != nil {
		return nil, fmt.Errorf("Provider readdir: %w", err)
	}
	sort.Slice(ents, func(i, j int) bool {
		return ents[i].Name < ents[j].Name
	})
	n.logger.Debugf("Read dir done, with %d entries, took %v", len(ents), time.Since(start))

	dirEnts := make([]fuse.DirEntry, len(ents))
	for i, gitEnt := range ents {
		dirEnts[i] = fuse.DirEntry{
			Mode: getEntryFileMode(gitEnt),
			Name: gitEnt.Name,
			Ino:  getEntryIno(gitEnt),
		}
	}

	n.subMu.Lock()
	defer n.subMu.Unlock()
	n.subDirEnts, n.subCache = dirEnts, true // cache it
	n.subEnts = ents

	return dirEnts, nil
}

func (n *Node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fusefs.Inode, syscall.Errno) {
	// lookup on memory nodes
	if cn := n.GetChild(name); cn != nil {
		switch subNode := cn.Operations().(type) {
		case *Node:
			subNode.entryToAttr(subNode.entry, &out.Attr)
		default:
			return nil, syscall.EIO
		}
		return cn, 0
	}

	_, err := n.listSubEntries(ctx)
	if err != nil {
		n.logger.Errorf("Ensure sub entries cache ready: %v", err)
		return nil, syscall.EIO
	}
	if !n.subCache {
		n.logger.Error("Unexpect error, the sub entries cache should be ready after ensuring")
		return nil, syscall.EIO
	}

	var found *types.Entry
	for _, ent := range n.subEnts {
		if ent.Name == name {
			found = ent
			break
		}
	}
	if found == nil {
		return nil, syscall.ENOENT
	}

	subNode := NewNode(found, n.provider)
	subAttr := subNode.entryToAttr(found, &out.Attr)
	return n.NewInode(ctx, subNode, subAttr), 0
}

func (n *Node) Open(ctx context.Context, flags uint32) (fh fusefs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	n.readContentMu.Lock()
	defer n.readContentMu.Unlock()

	if n.reader == nil {
		// The content of this file entry is empty, read from provider
		start := time.Now()
		data, err := n.provider.ReadFile(context.Background(), n.entry.Path)
		if err != nil {
			n.logger.Errorf("Read content from provider error: %v", err)
			return nil, 0, syscall.EIO
		}
		n.logger.Debugf("Download file done, size %s, took %v",
			humanize.Bytes(uint64(n.entry.Size)), time.Since(start))
		n.reader = bytes.NewReader(data)
	}

	return n, fuse.FOPEN_KEEP_CACHE, 0
}

func (n *Node) Getattr(ctx context.Context, f fusefs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	n.entryToAttr(n.entry, &out.Attr)
	return 0
}

func (n *Node) Getxattr(ctx context.Context, attr string, dest []byte) (uint32, syscall.Errno) {
	return 0, syscall.ENODATA
}

func (n *Node) Listxattr(ctx context.Context, dest []byte) (uint32, syscall.Errno) {
	return 0, 0
}

func (n *Node) Readlink(ctx context.Context) ([]byte, syscall.Errno) {
	return []byte(n.entry.LinkName), 0
}

func (n *Node) Statfs(ctx context.Context, out *fuse.StatfsOut) syscall.Errno {
	defaultStatfs(out)
	return 0
}

func (n *Node) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	readn, err := n.reader.ReadAt(dest, off)
	if err != nil && err != io.EOF {
		n.logger.Errorf("Read from content buffer error: %v", err)
		return nil, syscall.EIO
	}
	return fuse.ReadResultData(dest[:readn]), 0
}

func (n *Node) entryToAttr(ent *types.Entry, out *fuse.Attr) fusefs.StableAttr {
	ino := getEntryIno(ent)

	out.Ino = ino
	out.Size = uint64(ent.Size)
	if ent.IsSymLink {
		out.Size = uint64(len(ent.LinkName))
	}

	out.Blksize = blockSize
	out.Blocks = (out.Size + uint64(out.Blksize) - 1) / uint64(out.Blksize) * physicalBlockRatio

	out.SetTimes(nil, &n.createTime, nil)

	out.Mode = getEntryFileMode(ent)

	out.Owner = n.getOwner()

	return fusefs.StableAttr{
		Mode: out.Mode,
		Ino:  ino,
		// NOTE: The inode number is unique throughout the lifetime of
		// this filesystem so we don't consider about generation at this
		// moment.
	}
}

var (
	ownerInstance *fuse.Owner
	ownerOnce     sync.Once
)

func (n *Node) getOwner() fuse.Owner {
	ownerOnce.Do(func() {
		userInfo, err := user.Current()
		if err != nil {
			n.logger.Warnf("Get current user error: %v, we won't set owner info for entries", err)
			return
		}

		uid, err := strconv.Atoi(userInfo.Uid)
		if err != nil {
			n.logger.Warnf("User uid %q is not a number, you are not in a POSIX system?", userInfo.Uid)
			return
		}

		gid, err := strconv.Atoi(userInfo.Gid)
		if err != nil {
			n.logger.Warnf("User gid %q is not a number, you are not in a POSIX system?", userInfo.Gid)
			return
		}

		ownerInstance = &fuse.Owner{
			Uid: uint32(uid),
			Gid: uint32(gid),
		}
	})
	if ownerInstance == nil {
		return fuse.Owner{}
	}
	return *ownerInstance
}
