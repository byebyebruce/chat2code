package main

import (
	"fmt"
	"os"

	"github.com/byebyebruce/chat2code/vectordb/localdb"
	"github.com/byebyebruce/chat2code/version"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	apiKey  = os.Getenv("OPENAI_API_KEY")
	apiBase = os.Getenv("OPENAI_API_BASE")
)

func main() {
	fmt.Println(color.BlueString(version.Icon))
	fmt.Println("version:", color.GreenString(version.Version))

	rootCmd := RootCMD()
	rootCmd.AddCommand(
		LoadCMD(),
		DelCMD(),
		ListCMD(),
	)
	var (
		db string
	)

	rootCmd.PersistentFlags().StringVarP(&db, "db", "", `/tmp/chat2code.db`, "cache db")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		vDB, err := localdb.NewVectorSearch(db)
		if err != nil {
			return err
		}

		fmt.Println("load:", color.RedString(db))
		vecDB = vDB
		return nil
	}
	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		if vecDB != nil {
			fmt.Println("saving")
			vecDB.Close()
			fmt.Println("save ok!")
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
