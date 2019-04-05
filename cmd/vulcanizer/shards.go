package main

import (
	"fmt"
	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
	"os"
)

var nodesToCheck []string
var activeOnly bool

func init() {
	cmdShards.Flags().StringSliceVarP(&nodesToCheck, "nodes", "n", []string{}, "Elasticsearch node(s) to get shard information from")
	cmdShardsRecovery.Flags().StringSliceVarP(&nodesToCheck, "nodes", "n", []string{}, "Elasticsearch node(s) to get shard information from")
	cmdShardsRecovery.Flags().BoolVar(&activeOnly, "active", true, "Only display active recoveries")

	cmdShards.AddCommand(cmdShardsRecovery)
	rootCmd.AddCommand(cmdShards)
}

var cmdShards = &cobra.Command{
	Use:   "shards",
	Short: "Get shard data by cluster node(s).",
	Long:  `This command gets shard related data by node from the cluster.  Default is to return all shards.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port, auth := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		v.Auth = auth
		shards, err := v.GetShards(nodesToCheck)

		if err != nil {
			fmt.Printf("Error retrieving shard information: %s \n", err)
			os.Exit(1)
		}

		header := []string{"Index", "Shard", "Type", "State", "Docs", "Store", "IP", "Node"}
		rows := [][]string{}

		for _, shard := range shards {
			row := []string{
				shard.Index,
				shard.Shard,
				shard.Type,
				shard.State,
				shard.Docs,
				shard.Store,
				shard.IP,
				shard.Node,
			}
			rows = append(rows, row)
		}

		table := renderTable(rows, header)
		fmt.Println(table)
	},
}

var cmdShardsRecovery = &cobra.Command{
	Use:   "recovery",
	Short: "Get shard recovery status",
	Long:  `This command gets shard recovery status from the cluster.  Default is to return all shards.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port, auth := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		v.Auth = auth
		recovery, err := v.GetShardRecovery(nodesToCheck, activeOnly)

		if err != nil {
			fmt.Printf("Error retrieving recovery information: %s \n", err)
			os.Exit(1)
		}

		header := []string{"Index", "Shard", "Time", "Stage", "Source Node", "Target Node", "Bytes Percent", "Est Remaining"}
		var rows [][]string

		for _, shard := range recovery {
			remaining, _ := shard.TimeRemaining()

			row := []string{
				shard.Index,
				shard.Shard,
				shard.Time,
				shard.Stage,
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
