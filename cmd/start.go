package cmd

import (
	"errors"
	"os"
	"os/signal"

	"github.com/fioncat/grfs/fs"
	"github.com/fioncat/grfs/provider"
	"github.com/fioncat/grfs/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func Start() *cobra.Command {
	var path string
	var repo types.Repository
	var debug bool

	cmd := &cobra.Command{
		Use:   "start --host HOST --owner OWNER --name NAME [--target TARGET] [--github] [--debug]",
		Short: "Start the grfs fuse server",

		Args: cobra.ExactArgs(0),

		RunE: func(_ *cobra.Command, _ []string) error {
			if path == "" {
				return errors.New("Path could not be empty")
			}

			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}

			err := repo.Validate()
			if err != nil {
				return err
			}
			config, err := types.LoadConfig()
			if err != nil {
				return err
			}
			logrus.Debugf("The config value is: %+v", config)

			provider, err := provider.Load(&repo, config)
			if err != nil {
				return err
			}

			node := fs.NewNode(provider)
			fs, err := fs.Mount(node, path, config)
			if err != nil {
				return err
			}
			defer fs.Unmount()

			sigStop := make(chan os.Signal, 1)
			signal.Notify(sigStop, os.Interrupt)

			select {
			case <-sigStop:
				logrus.Info("Received interrupt signal, stop server")

			case <-fs.UnmountChan():
				logrus.Info("The grfs was unmountted by user, stop server")
			}

			return errors.New("Server stopped")
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&path, "path", "p", "", "The mount path")
	cmd.MarkFlagRequired("path")

	flags.StringVarP(&repo.Domain, "domain", "d", "", "The repo hostname")
	cmd.MarkFlagRequired("host")

	flags.StringVarP(&repo.Owner, "owner", "o", "", "The repo owner name")
	cmd.MarkFlagRequired("owner")

	flags.StringVarP(&repo.Name, "name", "n", "", "The repo name")
	cmd.MarkFlagRequired("name")

	flags.StringVarP(&repo.Ref, "ref", "r", "", "The repo ref")

	flags.BoolVarP(&debug, "debug", "", false, "Set log level to debug")

	return cmd
}
