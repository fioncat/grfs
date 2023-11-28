package storage

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/fioncat/grfs/osutils"
	"github.com/fioncat/grfs/types"
)

func buildTestMountPoint(i int) *types.MountPoint {
	return &types.MountPoint{
		Repo: &types.Repository{
			Domain: "github.com",
			Owner:  fmt.Sprintf("owner-%d", i),
			Name:   fmt.Sprintf("name-%d", i),
			Ref:    fmt.Sprintf("ref-%d", i),
		},
		Path:    fmt.Sprintf("path-%d", i),
		LogPath: fmt.Sprintf("log-%d", i),
	}
}

func TestBolt(t *testing.T) {
	err := osutils.EnsureDir("_test")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("_test/metadata.db")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	metadata, err := OpenBolt(&types.Config{
		BaseDir:         "_test",
		OpenBoltTimeout: time.Second * 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer metadata.Close()

	count := 30
	for i := 0; i < count; i++ {
		err = metadata.Put(buildTestMountPoint(i))
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < count; i++ {
		expect := buildTestMountPoint(i)
		var mp *types.MountPoint
		mp, err = metadata.Get(expect.Repo)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(mp, expect) {
			t.Fatalf("Unexpect mp from bolt: %+v, expect %+v", mp, expect)
		}
	}

	mps, err := metadata.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(mps) != count {
		t.Fatalf("Unexpect list count %d, expect %d", len(mps), count)
	}

	err = metadata.Remove(buildTestMountPoint(10))
	if err != nil {
		t.Fatal(err)
	}

	mps, err = metadata.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(mps) != count-1 {
		t.Fatalf("Unexpect list count %d, expect %d", len(mps), count-1)
	}

	_, err = metadata.Get(buildTestMountPoint(10).Repo)
	if !errors.Is(err, ErrMountPointNotFound) {
		t.Fatalf("Expect err to be not found, get: %v", err)
	}
}
