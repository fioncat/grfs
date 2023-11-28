package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/fioncat/grfs/storage"
	"github.com/fioncat/grfs/types"
	"github.com/spf13/cobra"
)

func Mount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mount [URL] [PATH]",
		Short: "Mount grfs to a path",

		Args: cobra.MaximumNArgs(2),
	}

	buildMountPointCommand(cmd, runMount)
	return cmd
}

func runMount(opts *MountPointOptions, args []string) error {
	if opts.Repo == nil {
		mps, err := opts.Metadata.List()
		if err != nil {
			return err
		}

		for _, mp := range mps {
			err = opts.mount(mp)
			if err != nil {
				return err
			}
		}

		return nil
	}

	if len(args) == 0 {
		mp, err := opts.Metadata.Get(opts.Repo)
		if err != nil {
			return err
		}

		return opts.mount(mp)
	}

	path := args[0]
	var err error
	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("convert path to abs: %w", err)
		}
	}

	mp, err := opts.Metadata.Get(opts.Repo)
	switch {
	case err == nil:
		if mp.Path != path {
			return fmt.Errorf("mountpoint %q has already been mounted on %q, please unmount it first", opts.Repo.String(), mp.Path)
		}
		return opts.mount(mp)

	case errors.Is(err, storage.ErrMountPointNotFound):
		var mps []*types.MountPoint
		mps, err = opts.Metadata.List()
		if err != nil {
			return fmt.Errorf("list mountpoint for path check: %w", err)
		}
		for _, mp := range mps {
			if mp.Path == path {
				return fmt.Errorf("path %q has already been mounted to %q, please use anthor path", path, mp.Repo.String())
			}
		}

		logDir := filepath.Join(opts.Config.BaseDir, "logs")
		mp, err = types.NewMountPoint(opts.Repo, path, logDir)
		if err != nil {
			return fmt.Errorf("create mountpoint: %w", err)
		}

		err = opts.mount(mp)
		if err != nil {
			return err
		}

		fmt.Printf("Put mountpoint %q to metadata\n", mp.Repo.String())
		err = opts.Metadata.Put(mp)
		if err != nil {
			return fmt.Errorf("put mountpoint to metadata: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("get mountpoint from metadata: %w", err)
	}

}
