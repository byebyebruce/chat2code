package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/byebyebruce/chat2code"
	"github.com/byebyebruce/chat2code/llm/openaillm"
	"github.com/byebyebruce/chat2code/vectordb/localdb"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

var (
	vecDB *localdb.VectorDatabase
	llm   *openaillm.OpenAILLM
)

func RootCMD() *cobra.Command {
	var (
		model                   string
		flagApiKey, flagApiBase string
	)

	rootCmd := &cobra.Command{
		Use: "chat2code",
	}

	rootCmd.Flags().StringVarP(&flagApiKey, "api-key", "k", "", "openai api key")
	rootCmd.Flags().StringVarP(&flagApiBase, "api-base", "b", "", "openai api base")
	rootCmd.Flags().StringVarP(&model, "model", "m", openai.GPT3Dot5Turbo, "model name(gpt-3.5-turbo/gpt-4/)")
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(flagApiKey) > 0 {
			apiKey = flagApiKey
		}
		if len(flagApiBase) > 0 {
			apiBase = flagApiBase
		}
		if len(apiKey) == 0 {
			fmt.Println("please set env OPENAI_API_KEY. if azure also need to set env OPENAI_API_BASE")
			return nil
		}
		t2c, err := chat2code.NewChat2Code(vecDB, openaillm.NewLLM(apiKey, apiBase, model))
		if err != nil {
			return err
		}

		repos, err := vecDB.ListRepos(cmd.Context())
		if err != nil {
			return err
		}
		if len(repos) == 0 {
			fmt.Println("there is no repo. use 'chat2code load {dir}'")
			return nil
		}

		prompt := promptui.Select{
			Label: "Select a repo to Q&A",
			Items: repos,
		}
		_, repo, err := prompt.Run()
		if err != nil {
			return nil
		}

		// loop
		for {
			fmt.Println()
			reader := bufio.NewReader(os.Stdin)

			color.Green("Enter question: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("error reading input: %v\n", err)
				fmt.Println()
				continue
			}
			input = strings.TrimSpace(input)
			if len(input) == 0 {
				continue
			}
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()

				answer, err := t2c.Answer(ctx, repo, input, 0.7)
				if err != nil {
					color.Red("err:%v", err)
					return
				}
				if answer == nil {
					color.Red("no answer")
					return
				}
				color.Cyan(answer.Answer)
				fmt.Printf(`"reference: %s"`+"\n", color.RedString(answer.File))
			}()
		}

		return nil
	}

	return rootCmd
}
