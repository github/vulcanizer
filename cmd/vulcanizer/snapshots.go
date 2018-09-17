package main

import (
	"fmt"
	"os"
	"time"

	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	cmdSnapshots.Flags().StringP("repository", "r", "", "Snapshot repository to query")
	err := cmdSnapshots.MarkFlagRequired("repository")
	if err != nil {
		fmt.Printf("Error binding repository configuration flag: %s \n", err)
		os.Exit(1)
	}
	rootCmd.AddCommand(cmdSnapshots)
}

var cmdSnapshots = &cobra.Command{
	Use:   "snapshots",
	Short: "Display the snapshots of the cluster.",
	Long:  `List the 10 most recent snapshots of the given repository`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)

		repository, err := cmd.Flags().GetString("repository")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: repository. Error: %s\n", err)
			os.Exit(1)
		}

		snapshots, err := v.GetSnapshots(repository)
		if err != nil {
			fmt.Printf("Could not query snapshots. Error: %s\n", err)
			os.Exit(1)
		}

		header := []string{"State", "Name", "Finished", "Duration"}

		if len(snapshots) > 10 {
			snapshots = snapshots[len(snapshots)-10:]
		}

		rows := [][]string{}
		for _, snapshot := range snapshots {
			duration, _ := time.ParseDuration(fmt.Sprintf("%dms", snapshot.DurationMillis))
			row := []string{
				snapshot.State,
				snapshot.Name,
				snapshot.EndTime.Format(time.RFC3339),
				fmt.Sprintf("%v", duration),
			}
			rows = append(rows, row)
		}

		fmt.Println(renderTable(rows, header))
	},
}
