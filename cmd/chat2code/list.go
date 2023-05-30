package main

import (
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func ListCMD() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "list repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			repos, err := vecDB.ListRepos(cmd.Context())
			if err != nil {
				return err
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"#", "Repo"})
			table.SetAutoWrapText(false)
			for i, repo := range repos {
				table.Append([]string{strconv.Itoa(i + 1), repo})
			}
			table.Render()

			return nil
		},
	}
	return listCmd
}
