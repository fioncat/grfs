package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/fioncat/grfs/cmd"
	"github.com/fioncat/grfs/types"
	"github.com/spf13/cobra"
)

var (
	Version     = "N/A"
	BuildType   = "N/A"
	BuildCommit = "N/A"
	BuildTime   = "N/A"
)

var rootCmd = &cobra.Command{
	Use: "grfs",

	Short: "The grfs command line tool",

	SilenceErrors: true,
	SilenceUsage:  true,

	Version: Version,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show csync full version info",

	Args: cobra.ExactArgs(0),

	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Printf("grpfs %s\n", Version)
		fmt.Printf("golang %s\n", strings.TrimPrefix(runtime.Version(), "go"))
		fmt.Println("")
		fmt.Printf("Build type:   %s\n", BuildType)
		fmt.Printf("Build target: %s-%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Commit SHA:   %s\n", BuildCommit)
		fmt.Printf("Build time:   %s\n", BuildTime)
		fmt.Println("")

		cfg, err := types.LoadConfig()
		if err != nil {
			return err
		}

		fmt.Printf("Config path: %s\n", cfg.Path)
		fmt.Printf("Base path:   %s\n", cfg.BaseDir)

		return nil
	},
}

func main() {
	rootCmd.AddCommand(cmd.Start())
	rootCmd.AddCommand(cmd.Mount())
	rootCmd.AddCommand(cmd.Unmount())
	rootCmd.AddCommand(cmd.Get())
	rootCmd.AddCommand(cmd.Logs())

	rootCmd.AddCommand(versionCmd)

	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("%s: %v\n", color.RedString("Error"), err)
		os.Exit(1)
	}
}
