package types

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

type tmpfsFilesystemMounter struct{}

func (m *tmpfsFilesystemMounter) Mount(mp *MountPoint) error {
	cmd := exec.Command("mount", "-t", "tmpfs", "-o", "size=100m", "tmpfs", mp.Path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount tmpfs failed: %w, output: %q", err, string(out))
	}
	return nil
}

func checkTestMountEnv(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.SkipNow()
	}

	uid := os.Getuid()
	if uid != 0 {
		t.Logf("Not root user, skip testing mount")
		t.SkipNow()
	}
}

func getTestMountpoint(dir string, t *testing.T) *MountPoint {
	checkTestMountEnv(t)

	fmt.Printf("Begin to test mount for %q\n", dir)
	repo, err := ParseRepository("github.com:fioncat/grfs")
	if err != nil {
		t.Fatalf("Parse repo: %v", err)
	}

	p, err := NewMountPoint(repo, dir+"mnt", dir+"logs")
	if err != nil {
		t.Fatalf("New mountpoint: %v", err)
	}

	return p
}

func TestMountpointMount(t *testing.T) {
	dir := "_test/mount/"
	p := getTestMountpoint(dir, t)
	err := p.Mount(&tmpfsFilesystemMounter{})
	if err != nil {
		t.Fatal(err)
	}

	status, errMsg := p.GetStatus()
	if status != MountPointStatusMounted {
		t.Fatalf("Invalid mountpoint status %q, errMsg: %q", status, errMsg)
	}

	// Try write and read
	msg := "Hello, world!"
	filename := filepath.Join(dir, "mnt", "test.txt")
	err = os.WriteFile(filename, []byte(msg), 0644)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != msg {
		t.Fatalf("Invalid read msg: %q", string(data))
	}

	err = p.Unmount()
	if err != nil {
		t.Fatal(err)
	}

	// After unmounting, the file should not be accessable
	_, err = os.Stat(filename)
	if !os.IsNotExist(err) {
		t.Fatalf("Expect %q not exists, err: %v", filename, err)
	}
}
