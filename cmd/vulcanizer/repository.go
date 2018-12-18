package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {

	cmdRepository.AddCommand(cmdRepositoryList)
	setupVerifySubCommand()

	rootCmd.AddCommand(cmdRepository)
}

var cmdRepository = &cobra.Command{
	Use:   "repository",
	Short: "Interact with the configured snapshot repositories.",
	Long:  `Use the list and verify subcommands.`,
}

func setupVerifySubCommand() {
	cmdRepositoryVerify.Flags().StringP("repository", "r", "", "Snapshot repository to query (required)")
	err := cmdRepositoryVerify.MarkFlagRequired("repository")
	if err != nil {
		fmt.Printf("Error binding repository configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdRepository.AddCommand(cmdRepositoryVerify)
}

var cmdRepositoryVerify = &cobra.Command{
	Use:   "verify",
	Short: "Verify the specified repository.",
	Long:  `This command will verify the repository is configured correctly on all nodes.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)

		repository, err := cmd.Flags().GetString("repository")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: repository. Error: %s\n", err)
			os.Exit(1)
		}

		verified, err := v.VerifyRepository(repository)

		if err != nil {
			fmt.Printf("Error verifying repository %s: %s\n", repository, err)
			os.Exit(1)
		}

		if verified {
			fmt.Printf("Repository %s is verified.\n", repository)
		} else {
			fmt.Printf("Repository %s is NOT verified.\n", repository)
		}
	},
}

var cmdRepositoryList = &cobra.Command{
	Use:   "list",
	Short: "List configured snapshot repositories.",
	Long:  `This command will list all the the snapshot repositories on the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)

		repos, err := v.GetRepositories()
		if err != nil {
			fmt.Printf("Error getting repositories. Error: %s\n", err)
			os.Exit(1)
		}

		header := []string{"Name", "Type", "Settings"}
		rows := [][]string{}

		for _, r := range repos {

			settings := []string{}

			for k, v := range r.Settings {
				settings = append(settings, fmt.Sprintf("%s: %v", k, v))
			}

			row := []string{
				r.Name,
				r.Type,
				strings.Join(settings, "\n"),
			}
			rows = append(rows, row)
		}

		fmt.Println(renderTable(rows, header))
	},
}
