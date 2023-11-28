package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/fioncat/grfs/types"
	"github.com/spf13/cobra"
)

func Logs() *cobra.Command {
	var lo logsOptions
	cmd := &cobra.Command{
		Use:   "logs [-f] [-n NUM] [--all] [URL]",
		Short: "Show fuse server logs",

		Args: cobra.MaximumNArgs(1),
	}

	buildMountPointCommand(cmd, lo.run)

	flags := cmd.Flags()
	flags.BoolVarP(&lo.all, "all", "a", false, "Print all logs")
	flags.BoolVarP(&lo.follow, "follow", "f", false, "Follow expand output")
	flags.IntVarP(&lo.number, "num", "n", 0, "tail number lines")

	return cmd
}

type logsOptions struct {
	all    bool
	follow bool
	number int
}

func (lo *logsOptions) run(opts *MountPointOptions, _ []string) error {
	var mp *types.MountPoint
	var err error
	if opts.Repo == nil {
		mps, err := opts.Metadata.List()
		if err != nil {
			return err
		}

		var lastMountPoint *types.MountPoint
		for _, mp := range mps {
			if lastMountPoint == nil {
				lastMountPoint = mp
				continue
			}
			if mp.CreateTime > lastMountPoint.CreateTime {
				lastMountPoint = mp
			}
		}

		if lastMountPoint == nil {
			return errors.New("no mountpoint, no log to display")
		}

		mp = lastMountPoint
	} else {
		mp, err = opts.Metadata.Get(opts.Repo)
		if err != nil {
			return fmt.Errorf("get mountpoint from metadata: %w", err)
		}
	}
	if lo.all {
		var file *os.File
		file, err = os.Open(mp.LogPath)
		if err != nil {
			return fmt.Errorf("open mountpoint log file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(os.Stdout, file)
		if err != nil {
			return fmt.Errorf("read mountpoint log file: %w", err)
		}

		return nil
	}

	var args []string
	if lo.follow {
		args = append(args, "-f")
	}
	if lo.number > 0 {
		args = append(args, "-n", fmt.Sprint(lo.number))
	}
	args = append(args, mp.LogPath)
	cmd := exec.Command("tail", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("tail command exited: %w", err)
	}

	return nil
}
