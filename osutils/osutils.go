package osutils

import (
	"fmt"
	"os"
	"path/filepath"
)

func EnsureDir(dir string) error {
	stat, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, os.ModePerm)
		}
		return err
	}

	if !stat.IsDir() {
		return fmt.Errorf("%q is not a directory", dir)
	}

	return nil
}

func EnsureFilePathDir(filename string) error {
	dir := filepath.Dir(filename)
	return EnsureDir(dir)
}
