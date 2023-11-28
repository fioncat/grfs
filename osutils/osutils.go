package osutils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/olekukonko/tablewriter"
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

func ShowTable(titles []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(titles)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("  ") // pad with tabs
	table.SetNoWhiteSpace(true)
	table.AppendBulk(rows) // Add Bulk Data
	table.Render()
}
