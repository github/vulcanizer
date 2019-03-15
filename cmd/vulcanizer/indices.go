package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cmdIndices)
}

var cmdIndices = &cobra.Command{
	Use:   "indices",
	Short: "Display the indices of the cluster.",
	Long:  `Show what indices are created on the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		indices, err := v.GetIndices()

		if err != nil {
			fmt.Printf("Error getting indices: %s\n", err)
			os.Exit(1)
		}

		header := []string{"Health", "Status", "Name", "Primary Shards", "Replica Count", "Index", "Docs"}
		rows := [][]string{}

		for _, index := range indices {
			row := []string{
				index.Health,
				index.Status,
				index.Name,
				strconv.Itoa(index.PrimaryShards),
				strconv.Itoa(index.ReplicaCount),
				index.IndexSize,
				strconv.Itoa(index.DocumentCount),
			}

			rows = append(rows, row)
		}

		table := renderTable(rows, header)
		fmt.Println(table)
	},
}
