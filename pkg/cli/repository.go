package cli

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
	setupRegisterSubCommand()
	setupRemoveSubCommand()

	rootCmd.AddCommand(cmdRepository)
}

var cmdRepository = &cobra.Command{
	Use:   "repository",
	Short: "Interact with the configured snapshot repositories.",
	Long:  `Use the list, verify, remove, and register subcommands.`,
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

func setupRemoveSubCommand() {
	cmdRepositoryRemove.Flags().StringP("repository", "r", "", "Snapshot repository to remove (required)")
	err := cmdRepositoryRemove.MarkFlagRequired("repository")
	if err != nil {
		fmt.Printf("Error binding repository configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdRepository.AddCommand(cmdRepositoryRemove)
}

func setupRegisterSubCommand() {
	cmdRepositoryRegister.Flags().StringP("repository", "r", "", "Snapshot repository name to register (required)")
	err := cmdRepositoryRegister.MarkFlagRequired("repository")
	if err != nil {
		fmt.Printf("Error binding repository configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdRepositoryRegister.Flags().StringP("type", "t", "", "Type of snapshot repository to register (required)")
	err = cmdRepositoryRegister.MarkFlagRequired("type")
	if err != nil {
		fmt.Printf("Error binding repository configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdRepositoryRegister.Flags().StringToStringP("settings", "s", map[string]string{}, "Settings of the repository to register in key value pairs, i.e. location=/backups,compress=true")

	cmdRepository.AddCommand(cmdRepositoryRegister)
}

var cmdRepositoryVerify = &cobra.Command{
	Use:   "verify",
	Short: "Verify the specified repository.",
	Long:  `This command will verify the repository is configured correctly on all nodes.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

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

		v := getClient()

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

var cmdRepositoryRegister = &cobra.Command{
	Use:   "register",
	Short: "Register specified repository.",
	Long:  `This command will register a snapshot repository.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		repositoryName, err := cmd.Flags().GetString("repository")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: repository. Error: %s\n", err)
			os.Exit(1)
		}

		repoType, err := cmd.Flags().GetString("type")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: type. Error: %s\n", err)
			os.Exit(1)
		}

		stringSettings, err := cmd.Flags().GetStringToString("settings")
		if err != nil {
			fmt.Printf("Could not retrieve argument: settings. Error: %s\n", err)
			os.Exit(1)
		}

		settings := make(map[string]interface{}, len(stringSettings))
		for k, v := range stringSettings {
			settings[k] = v
		}

		repository := vulcanizer.Repository{
			Name:     repositoryName,
			Type:     repoType,
			Settings: settings,
		}

		err = v.RegisterRepository(repository)

		if err != nil {
			fmt.Printf("Error registering repository %s: %s\n", repositoryName, err)
			os.Exit(1)
		}

		fmt.Printf("Repository %s registered successfully.\n", repositoryName)
	},
}

var cmdRepositoryRemove = &cobra.Command{
	Use:   "remove",
	Short: "Remove specified repository.",
	Long:  `This command will remove a snapshot repository.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		repositoryName, err := cmd.Flags().GetString("repository")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: repository. Error: %s\n", err)
			os.Exit(1)
		}

		err = v.RemoveRepository(repositoryName)

		if err != nil {
			fmt.Printf("Error removing repository %s: %s\n", repositoryName, err)
			os.Exit(1)
		}

		fmt.Printf("Repository %s removed successfully.\n", repositoryName)
	},
}
