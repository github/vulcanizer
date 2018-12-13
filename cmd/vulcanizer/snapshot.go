package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/github/vulcanizer"
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

	cmdSnapshotRestore.Flags().StringP("snapshot", "s", "", "Snapshot name to query (required)")
	err = cmdSnapshotRestore.MarkFlagRequired("snapshot")
	if err != nil {
		fmt.Printf("Error binding snapshot configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdSnapshotRestore.Flags().StringP("repository", "r", "", "Snapshot repository to query (required)")
	err = cmdSnapshotRestore.MarkFlagRequired("repository")
	if err != nil {
		fmt.Printf("Error binding repository configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdSnapshotRestore.Flags().StringP("prefix", "", "restored_", "What to prefix on the restored index")
	err = cmdSnapshotRestore.MarkFlagRequired("prefix")
	if err != nil {
		fmt.Printf("Error binding prefix configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdSnapshotRestore.Flags().StringP("index", "i", "", "Which index to restore from the snapshot")
	err = cmdSnapshotRestore.MarkFlagRequired("index")
	if err != nil {
		fmt.Printf("Error binding index configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdSnapshot.AddCommand(cmdSnapshotRestore)

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
		v := vulcanizer.NewClient(host, port)

		snapshotName, err := cmd.Flags().GetString("snapshot")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: snapshot. Error: %s\n", err)
			os.Exit(1)
		}

		repository, err := cmd.Flags().GetString("repository")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: repository. Error: %s\n", err)
			os.Exit(1)
		}

		snapshot, err := v.GetSnapshotStatus(repository, snapshotName)
		if err != nil {
			fmt.Printf("Error getting snapshot. Error: %s\n", err)
			os.Exit(1)
		}

		duration, _ := time.ParseDuration(fmt.Sprintf("%dms", snapshot.DurationMillis))

		results := [][]string{
			[]string{"State", snapshot.State},
			[]string{"Name", snapshot.Name},
			[]string{"Indices", strings.Join(snapshot.Indices, ", ")},
			[]string{"Started", snapshot.StartTime.Format(time.RFC3339)},
			[]string{"Finished", snapshot.EndTime.Format(time.RFC3339)},
			[]string{"Duration", fmt.Sprintf("%v", duration)},
			[]string{"Shards", fmt.Sprintf("Successful shards: %d, failed shards: %d", snapshot.Shards.Successful, snapshot.Shards.Failed)},
		}

		fmt.Println(renderTable(results, []string{"Metric", "Value"}))
	},
}

var cmdSnapshotRestore = &cobra.Command{
	Use:   "restore",
	Short: "Restore a snapshot.",
	Long:  `This command will restore a specific index from a snapshot to the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)

		snapshotName, err := cmd.Flags().GetString("snapshot")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: snapshot. Error: %s\n", err)
			os.Exit(1)
		}

		repository, err := cmd.Flags().GetString("repository")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: repository. Error: %s\n", err)
			os.Exit(1)
		}

		prefix, err := cmd.Flags().GetString("prefix")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: prefix. Error: %s\n", err)
			os.Exit(1)
		}

		index, err := cmd.Flags().GetString("index")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: index. Error: %s\n", err)
			os.Exit(1)
		}

		err = v.RestoreSnapshotIndices(repository, snapshotName, []string{index}, prefix)
		if err != nil {
			fmt.Printf("Error while calling restore snapshot API. Error: %s\n", err)
			os.Exit(1)
		}

		fmt.Println("Restore operation called successfully.")
	},
}
