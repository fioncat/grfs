package types

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	path := "_test/not_exists.yaml"
	basedir := "_test/default_basedir"
	// The config file not exists, will use default config
	os.Setenv("GRFS_CONFIG_PATH", path)
	os.Setenv("GRFS_BASE_PATH", basedir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	expect := newDefaultConfig(path, basedir)

	if !reflect.DeepEqual(cfg, expect) {
		t.Fatalf("Unexpect config %+v, expect %+v", cfg, expect)
	}
}

const testConfigYaml = `
openBoltTimeout: "20s"
fs:
  allowOthers: true
  entryTimeout: "120s"
  debug: true
auths:
  github.com: "test-github-token"
  gitlab.com: "test-gitlab-token"
`

var testExpectConfig = &Config{
	OpenBoltTimeout: time.Second * 20,

	Fs: &FilesystemConfig{
		AllowOthers:  true,
		EntryTimeout: time.Minute * 2,
		Debug:        true,
	},

	Auths: Auths{
		"github.com": "test-github-token",
		"gitlab.com": "test-gitlab-token",
	},
}

func TestLoadConfig(t *testing.T) {
	path := "_test/config.yaml"
	basedir := "_test/basedir"

	os.Setenv("GRFS_CONFIG_PATH", path)
	os.Setenv("GRFS_BASE_PATH", basedir)

	err := os.WriteFile(path, []byte(testConfigYaml), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	testExpectConfig.Path = path
	testExpectConfig.BaseDir = basedir

	if !reflect.DeepEqual(cfg, testExpectConfig) {
		t.Fatalf("Unexpect config %+v, expect %+v", cfg, testExpectConfig)
	}
}
