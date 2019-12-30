package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/leosunmo/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cmdIndices)
	cmdIndices.AddCommand(cmdOpen)
	cmdIndices.AddCommand(cmdClose)
}

var cmdIndices = &cobra.Command{
	Use:     "indices",
	Aliases: []string{"index"},
	Short:   "Display the indices of the cluster.",
	Long:    `Show what indices are created on the given cluster.`,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		var err error
		var indices []vulcanizer.Index
		if len(args) > 0 {
			indices, err = v.GetIndices(args[0])
		} else {
			indices, err = v.GetAllIndices()
		}

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

var cmdOpen = &cobra.Command{
	Use:   "open",
	Short: "Open the given index/indices",
	Long:  `Given a name or pattern, opens the index/indices`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		err := v.OpenIndex(args[0])
		if err != nil {
			fmt.Printf("Error opening index/indices: %s - %s\n", args[0], err)
			os.Exit(1)
		}
	},
}

var cmdClose = &cobra.Command{
	Use:   "close",
	Short: "Close the given index/indices",
	Long:  `Given a name or pattern, closes the index/indices`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		err := v.CloseIndex(args[0])
		if err != nil {
			fmt.Printf("Error opening index/indices: %s - %s\n", args[0], err)
			os.Exit(1)
		}
	},
}
