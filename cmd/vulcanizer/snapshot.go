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
	setupStatusSubCommand()
	setupRestoreSubCommand()
	setupListSubCommand()
	setupCreateSubCommand()

	rootCmd.AddCommand(cmdSnapshot)
}

var cmdSnapshot = &cobra.Command{
	Use:   "snapshot",
	Short: "Interact with a specific snapshot.",
	Long:  `Use the status, list, and restore subcommands.`,
}

func setupStatusSubCommand() {
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
}

func setupCreateSubCommand() {
	cmdSnapshotCreate.Flags().StringP("snapshot", "s", "", "Snapshot name to query (required)")
	err := cmdSnapshotCreate.MarkFlagRequired("snapshot")
	if err != nil {
		fmt.Printf("Error binding snapshot configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdSnapshotCreate.Flags().StringP("repository", "r", "", "Snapshot repository to query (required)")
	err = cmdSnapshotCreate.MarkFlagRequired("repository")
	if err != nil {
		fmt.Printf("Error binding repository configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdSnapshotCreate.Flags().BoolP("all-indices", "a", false, "Snapshot all indices on the cluster. Takes precedence over index arguments.")

	cmdSnapshotCreate.Flags().StringSliceP("index", "i", []string{}, "Snapshot specific indices on the cluster. Can be repeated.")

	cmdSnapshot.AddCommand(cmdSnapshotCreate)
}

func setupListSubCommand() {
	cmdSnapshotList.Flags().StringP("repository", "r", "", "Snapshot repository to query (required)")
	err := cmdSnapshotList.MarkFlagRequired("repository")
	if err != nil {
		fmt.Printf("Error binding repository configuration flag: %s \n", err)
		os.Exit(1)
	}
	cmdSnapshot.AddCommand(cmdSnapshotList)
}

func setupRestoreSubCommand() {
	cmdSnapshotRestore.Flags().StringP("snapshot", "s", "", "Snapshot name to query (required)")
	err := cmdSnapshotRestore.MarkFlagRequired("snapshot")
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
			{"State", snapshot.State},
			{"Name", snapshot.Name},
			{"Indices", strings.Join(snapshot.Indices, ", ")},
			{"Started", snapshot.StartTime.Format(time.RFC3339)},
			{"Finished", snapshot.EndTime.Format(time.RFC3339)},
			{"Duration", fmt.Sprintf("%v", duration)},
			{"Shards", fmt.Sprintf("Successful shards: %d, failed shards: %d", snapshot.Shards.Successful, snapshot.Shards.Failed)},
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

var cmdSnapshotList = &cobra.Command{
	Use:   "list",
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

var cmdSnapshotCreate = &cobra.Command{
	Use:   "create",
	Short: "Create a new snapshot.",
	Long:  `This command will take a new snapshot of the data of either all or specified indices.`,
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

		allIndices, err := cmd.Flags().GetBool("all-indices")
		if err != nil {
			fmt.Printf("Could not retrieve argument: all-indices. Error: %s\n", err)
			os.Exit(1)
		}

		indices, err := cmd.Flags().GetStringSlice("index")
		if err != nil {
			fmt.Printf("Could not retrieve argument: index. Error: %s\n", err)
			os.Exit(1)
		}

		if allIndices {
			err = v.SnapshotAllIndices(repository, snapshotName)
			if err != nil {
				fmt.Printf("Error while taking snapshot. Error: %s\n", err)
				os.Exit(1)
			}
			fmt.Println("Snapshot operation started.")
		} else {
			if len(indices) == 0 {
				fmt.Printf("Got 0 indices to snapshot. Please specify indices with `--index` or all indices with `--all-indices`.\n")
				os.Exit(1)
			}

			err = v.SnapshotIndices(repository, snapshotName, indices)
			if err != nil {
				fmt.Printf("Error while taking snapshot. Error: %s\n", err)
				os.Exit(1)
			}

			fmt.Println("Snapshot operation started.")
		}
	},
}
