package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func DelCMD() *cobra.Command {
	var deleteCmd = &cobra.Command{
		Use: "delete",

		Short: "delete a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			repos, err := vecDB.ListRepos(cmd.Context())
			if err != nil {
				return err
			}
			if len(repos) == 0 {
				fmt.Println("no repo")
				return nil
			}

			sel := promptui.Select{
				Label: "Select a repo to delete",
				Items: repos,
			}
			_, result, err := sel.Run()
			if err != nil {
				return nil
			}
			prompt := promptui.Prompt{
				Label:     "Delete [" + result + "]",
				IsConfirm: true,
			}

			ok, err := prompt.Run()
			if err != nil {
				return nil
			}

			if ok != "y" {
				return nil
			}
			if err != vecDB.DeleteRepo(cmd.Context(), result) {
				return err
			}
			color.Red("Repo [" + result + "] has been deleted")

			return nil
		},
	}
	return deleteCmd
}
