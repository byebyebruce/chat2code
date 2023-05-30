package main

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/byebyebruce/chat2code"
	"github.com/byebyebruce/chat2code/llm/openaillm"
	"github.com/byebyebruce/chat2code/pkg/splitter/text_splitter"
	"github.com/byebyebruce/chat2code/pkg/util"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func LoadCMD() *cobra.Command {
	var (
		filter                  string
		flagApiKey, flagApiBase string
	)

	loadCmd := &cobra.Command{
		Use:   "load dir",
		Short: "load a dir",
	}
	loadCmd.Flags().StringVarP(&flagApiKey, "api-key", "k", "", "openai api key")
	loadCmd.Flags().StringVarP(&flagApiBase, "api-base", "b", "", "openai api base")
	loadCmd.Flags().StringVarP(&filter, "filter", "f", `.*\.go$`, "filter regex")
	loadCmd.RunE = func(cmd *cobra.Command, args []string) error { // openai client
		if len(flagApiKey) > 0 {
			apiKey = flagApiKey
		}
		if len(flagApiBase) > 0 {
			apiKey = flagApiBase
		}
		if len(apiKey) == 0 {
			fmt.Println("please set env OPENAI_API_KEY,if azure also need set env OPENAI_API_BASE")
			return nil
		}
		t2c, err := chat2code.NewChat2Code(vecDB, openaillm.NewLLM(apiKey, apiBase, ""))
		if err != nil {
			return err
		}

		// 文件目录
		dirs := args
		if len(args) == 0 {
			fmt.Println("chat2code load dir1 dir2 ...")
			return nil
		}

		load := func(dir string) error {
			repo, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			// 分析目录
			fs, err := util.WalkDir(repo, regexp.MustCompile(filter))
			if err != nil {
				return err
			}
			if len(fs) == 0 {
				return nil
			}

			// load chunks
			chunks, err := text_splitter.SplitFiles(fs, 4096, 0)
			if err != nil {
				fmt.Println(err)
				return nil
			}
			fmt.Println(color.GreenString(repo), color.RedString("files:%d", len(fs)), color.RedString("chunks:%d", len(chunks)))

			// create repo
			if err := vecDB.CreateIfNotExists(repo); err != nil {
				return err
			}

			// load
			return t2c.Load(context.Background(), repo, chunks, 2)
		}
		for _, dir := range dirs {
			if err := load(dir); err != nil {
				return err
			}
		}
		return nil
	}

	return loadCmd
}
