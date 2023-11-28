package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fioncat/grfs/osutils"
	"github.com/fioncat/grfs/types"
	"github.com/spf13/cobra"
)

func Get() *cobra.Command {
	var showJson bool
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show mountpoint",

		Args: cobra.MaximumNArgs(1),
	}
	buildMountPointCommand(cmd, runGet(&showJson))

	cmd.Flags().BoolVarP(&showJson, "json", "J", false, "Show json output")

	return cmd
}

func runGet(showJson *bool) func(opts *MountPointOptions, args []string) error {
	return func(opts *MountPointOptions, args []string) error {
		items, err := getMountpointItems(opts)
		if err != nil {
			return err
		}

		if *showJson {
			data, err := json.MarshalIndent(items, "", "  ")
			if err != nil {
				return fmt.Errorf("Marshal json items: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		if len(items) == 0 {
			fmt.Println("No mountpoint")
			return nil
		}

		rows := make([][]string, len(items))
		for i, item := range items {
			status := item.Status.Color()
			rows[i] = []string{
				item.Repo.String(),
				status,
				item.Path,
			}
		}

		osutils.ShowTable([]string{"Repository", "Status", "Path"}, rows)
		return nil
	}
}

func getMountpointItems(opts *MountPointOptions) ([]*types.MountPointDisplay, error) {
	if opts.Repo != nil {
		mp, err := opts.Metadata.Get(opts.Repo)
		if err != nil {
			return nil, fmt.Errorf("get mountpoint from metadata: %w", err)
		}
		return []*types.MountPointDisplay{mp.Display()}, nil
	}

	items, err := opts.Metadata.List()
	if err != nil {
		return nil, err
	}

	displayItems := make([]*types.MountPointDisplay, len(items))
	for i, item := range items {
		displayItems[i] = item.Display()
	}

	return displayItems, nil
}
