package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var nodesToCheckRecovery []string

func init() {
	cmdShardRecovery.Flags().StringArrayVarP(&nodesToCheckRecovery, "nodes", "n", []string{}, "Elasticsearch nodes to view shard recovery progress (optional, omitted will include all nodes)")
	rootCmd.AddCommand(cmdShardRecovery)
}

var cmdShardRecovery = &cobra.Command{
	Use:   "recovery",
	Short: "Display the recovery progress of shards.",
	Long:  `Show the details regarding shard recovery operations across a set of cluster nodes.`,
	Run: func(cmd *cobra.Command, args []string) {
		v := getClient()

		recovery, err := v.GetShardRecovery(nodesToCheckRecovery, true)

		if err != nil {
			fmt.Printf("Error getting shard recovery details: %s\n", err)
			os.Exit(1)
		}
		header := []string{"Index", "Shard", "Time", "Source", "Target", "Bytes %", "Est Remaining"}
		var rows [][]string

		for _, shard := range recovery {
			remaining, _ := shard.TimeRemaining()
			row := []string{
				shard.Index,
				shard.Shard,
				shard.Time,
				shard.SourceNode,
				shard.TargetNode,
				shard.BytesPercent,
				remaining.String(),
			}
			rows = append(rows, row)
		}

		table := renderTable(rows, header)

		fmt.Println(table)
	},
}
