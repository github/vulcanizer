package main

import (
	"fmt"
	"os"

	v "github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	cmdSnapshots.Flags().StringP("repository", "r", "", "Snapshot repository to query")
	err := cmdSnapshots.MarkFlagRequired("repository")
	if err != nil {
		panic(err)
	}
	rootCmd.AddCommand(cmdSnapshots)
}

var cmdSnapshots = &cobra.Command{
	Use:   "snapshots",
	Short: "Display the snapshots of the cluster.",
	Long:  `List the 10 most recent snapshots of the given repository`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()

		repository, err := cmd.Flags().GetString("repository")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: repository. Error: %s\n", err)
			os.Exit(1)
		}

		rows, headers := v.GetSnapshots(host, port, repository)

		fmt.Println(renderTable(rows, headers))
	},
}
