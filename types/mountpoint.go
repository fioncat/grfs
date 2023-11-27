package types

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/fioncat/grfs/osutils"
	"k8s.io/mount-utils"
)

type MountPointStatus string

const (
	waitMountPointReadyTimeout  = time.Second * 3
	waitMountPointReadyInterval = time.Millisecond * 100
)

const (
	MountPointStatusMounted   = "mounted"
	MountPointStatusUnmounted = "unmounted"
	MountPointStatusLost      = "lost"
	MountPointStatusError     = "error"
)

func (s MountPointStatus) Color() string {
	switch s {
	case MountPointStatusMounted:
		return color.GreenString(string(s))

	case MountPointStatusUnmounted:
		return color.YellowString(string(s))

	case MountPointStatusLost, MountPointStatusError:
		return color.RedString(string(s))
	}
	return ""
}

type MountPoint struct {
	Repo *Repository `json:"repo"`

	Path string `json:"path"`

	LogPath string `json:"logPath"`
}

type MountPointDisplay struct {
	MountPoint

	Status MountPointStatus `json:"status,omitempty"`

	ErrorMessage string `json:"errMsg,omitempty"`
}

type FilesystemMounter interface {
	Mount(mp *MountPoint) error
}

type MountPointMetadata interface {
	Put(mp *MountPoint) error
	Get(repo *Repository) (*MountPoint, error)
	List() ([]*MountPoint, error)
	Remove(mp *MountPoint) error
}

func NewMountPoint(repo *Repository, path, logDir string) (*MountPoint, error) {
	repoPath := strings.ReplaceAll(repo.String(), ":", "/")
	logPath := filepath.Join(logDir, repoPath)
	err := osutils.EnsureFilePathDir(logPath)
	if err != nil {
		return nil, err
	}

	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("convert mountpoint to abs: %w", err)
		}
	}

	return &MountPoint{
		Repo:    repo,
		Path:    path,
		LogPath: logPath,
	}, nil
}

func (mp *MountPoint) GetStatus() (MountPointStatus, string) {
	ismount, err := mp.newMounter().IsMountPoint(mp.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return MountPointStatusUnmounted, ""
		}

		// For the FUSE filesystem, if the daemon exits abnormally and the mount
		// point is not cleaned, accessing the mount point will return ENOTCONN
		// error. In this case, return `Lost` status.
		syserr := errors.Unwrap(err)
		if errors.Is(syserr, syscall.ENOTCONN) {
			return MountPointStatusLost, ""
		}

		// The FUSE filesystem has error.
		return MountPointStatusError, err.Error()
	}

	if ismount {
		return MountPointStatusMounted, ""
	}
	return MountPointStatusUnmounted, ""
}

func (mp *MountPoint) Mount(fsMounter FilesystemMounter) error {
	mounter := mp.newMounter()
	status, _ := mp.GetStatus()

	switch status {
	case MountPointStatusMounted:
		return nil

	case MountPointStatusUnmounted:
		err := osutils.EnsureDir(mp.Path)
		if err != nil {
			return fmt.Errorf("ensure mountpoint: %w", err)
		}

	case MountPointStatusLost, MountPointStatusError:
		// For Lost mountpoint, the fuse server may be killed or exited abnormally.
		// For Error mountpoint, the fuse server may have error(s).
		// No matter what the situation is, we should try to unmount first, the
		// kernel should automatically clean up the abnormal state.
		err := mounter.Unmount(mp.Path)
		if err != nil {
			return fmt.Errorf("unmount %s mountpoint: %w", status, err)
		}
	}

	ents, err := os.ReadDir(mp.Path)
	if err != nil {
		return fmt.Errorf("read mountpoint: %w", err)
	}
	if len(ents) > 0 {
		return fmt.Errorf("mountpoint path %q is not empty, cannot be mounted", mp.Path)
	}

	err = fsMounter.Mount(mp)
	if err != nil {
		return err
	}

	waitReadyTicker := time.NewTicker(waitMountPointReadyInterval)
	waitReadyTimeoutTimer := time.NewTimer(waitMountPointReadyTimeout)

	for {
		select {
		case <-waitReadyTicker.C:
			status, _ = mp.GetStatus()
			if status == MountPointStatusMounted {
				return nil
			}

		case <-waitReadyTimeoutTimer.C:
			status, _ = mp.GetStatus()
			return fmt.Errorf("wait mountpoint ready timeout after %v, status is: %q, please check fs log file: %q", waitMountPointReadyTimeout, status, mp.LogPath)
		}
	}
}

func (mp *MountPoint) Unmount() error {
	status, _ := mp.GetStatus()
	if status != MountPointStatusUnmounted {
		err := mp.newMounter().Unmount(mp.Path)
		if err != nil {
			return err
		}
	}

	// The mountpoint should be empty, it is safe to call `os.Remove` here.
	// If mountpoint has file(s), the method would fail, the user should
	// handle it.
	err := os.Remove(mp.Path)
	if err != nil {
		return fmt.Errorf("remove mountpoint: %w", err)
	}

	return nil
}

func (mp *MountPoint) Display() *MountPointDisplay {
	status, errMsg := mp.GetStatus()
	return &MountPointDisplay{
		MountPoint:   *mp,
		Status:       status,
		ErrorMessage: errMsg,
	}
}

func (mp *MountPoint) newMounter() mount.Interface {
	return mount.New(os.Getenv("GRFS_MOUNT_PATH"))
}
