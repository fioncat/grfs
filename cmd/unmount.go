package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Unmount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unmount [URL]",
		Short: "Unmount grfs",

		Args: cobra.MaximumNArgs(1),
	}

	buildMountPointCommand(cmd, runUnmount)
	return cmd
}

func runUnmount(opts *MountPointOptions, _ []string) error {
	if opts.Repo == nil {
		mps, err := opts.Metadata.List()
		if err != nil {
			return err
		}

		for _, mp := range mps {
			err = opts.unmount(mp)
			if err != nil {
				return err
			}
		}

		return nil
	}

	mp, err := opts.Metadata.Get(opts.Repo)
	if err != nil {
		return fmt.Errorf("get mountpoint in metadata: %w", err)
	}

	return opts.unmount(mp)
}
