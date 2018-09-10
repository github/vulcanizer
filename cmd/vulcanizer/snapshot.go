package main

import (
	"fmt"
	"os"

	v "github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	cmdSnapshotStatus.Flags().StringP("snapshot", "s", "", "Snapshot name to query (required)")
	err := cmdSnapshotStatus.MarkFlagRequired("snapshot")
	if err != nil {
		fmt.Printf("Error binding snapshot configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdSnapshotStatus.Flags().StringP("repository", "r", "", "Snapshot repository to query (required)")
	err = cmdSnapshotStatus.MarkFlagRequired("repository")
	if err != nil {
		fmt.Printf("Error binding repository configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdSnapshot.AddCommand(cmdSnapshotStatus)
	rootCmd.AddCommand(cmdSnapshot)
}

var cmdSnapshot = &cobra.Command{
	Use:   "snapshot",
	Short: "Interact with a specific snapshot.",
	Long:  `Use the status subcommand to show detailed information about a snapshot.`,
}

var cmdSnapshotStatus = &cobra.Command{
	Use:   "status",
	Short: "Display info about a snapshot.",
	Long:  `This command will display detailed information about the given snapshot.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()

		snapshot, err := cmd.Flags().GetString("snapshot")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: snapshot. Error: %s\n", err)
			os.Exit(1)
		}

		repository, err := cmd.Flags().GetString("repository")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: repository. Error: %s\n", err)
			os.Exit(1)
		}

		rows, headers := v.GetSnapshotStatus(host, port, repository, snapshot)
		fmt.Println(renderTable(rows, headers))
	},
}
