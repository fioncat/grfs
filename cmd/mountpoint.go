package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/fioncat/grfs/provider"
	"github.com/fioncat/grfs/storage"
	"github.com/fioncat/grfs/types"
	"github.com/spf13/cobra"
)

type grfsMounter struct {
	debug bool
}

func (m *grfsMounter) Mount(mp *types.MountPoint) error {
	args := []string{
		"start",
		"--path", mp.Path,
		"--domain", mp.Repo.Domain,
		"--owner", mp.Repo.Owner,
		"--name", mp.Repo.Name,
	}
	if mp.Repo.Ref != "" {
		args = append(args, "--ref", mp.Repo.Ref)
	}
	if m.debug {
		args = append(args, "--debug")
	}

	logFile, err := os.OpenFile(mp.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open mountpoint log file: %w", err)
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start daemon process: %w", err)
	}

	return nil
}

type MountPointOptions struct {
	Config *types.Config

	Metadata types.MountPointMetadata

	Repo *types.Repository

	mounter types.FilesystemMounter
}

func buildMountPointCommand(cmd *cobra.Command, action func(opts *MountPointOptions, args []string) error) {
	cmd.RunE = func(_ *cobra.Command, args []string) error {
		cfg, err := types.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		metadata, err := storage.OpenBolt(cfg)
		if err != nil {
			return fmt.Errorf("open metadata database: %w", err)
		}
		defer metadata.Close()

		var repo *types.Repository
		if len(args) >= 1 {
			url := args[0]
			repo, err = types.ParseRepository(url)
			if err != nil {
				return fmt.Errorf("parse repo: %w", err)
			}

			prov, err := provider.Load(repo, cfg)
			if err != nil {
				return fmt.Errorf("load provider: %w", err)
			}
			err = prov.Check(context.Background())
			if err != nil {
				return fmt.Errorf("check repository: %w", err)
			}

			args = args[1:]
		}

		opts := &MountPointOptions{
			Config:   cfg,
			Metadata: metadata,
			Repo:     repo,
			mounter:  &grfsMounter{debug: cfg.Fs.Debug},
		}
		return action(opts, args)
	}
}

func (opts *MountPointOptions) mount(mp *types.MountPoint) error {
	err := mp.Mount(opts.mounter)
	if err != nil {
		return fmt.Errorf("mount %q on %q: %w", mp.Repo.String(), mp.Path, err)
	}
	fmt.Printf("Mounted grfs: %q no %q\n", mp.Repo.String(), mp.Path)
	return nil
}

func (opts *MountPointOptions) unmount(mp *types.MountPoint) error {
	err := mp.Unmount()
	if err != nil {
		return fmt.Errorf("unmount %q: %w", mp.Repo.String(), err)
	}
	err = opts.Metadata.Remove(mp)
	if err != nil && !errors.Is(err, storage.ErrMountPointNotFound) {
		return fmt.Errorf("remove mountpoint in metadata: %w", err)
	}

	fmt.Printf("Unmounted grfs: %q\n", mp.Repo.String())
	return nil
}
