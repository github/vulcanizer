package main

import (
	"fmt"
	"os"

	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cmdNodes)
}

var cmdNodes = &cobra.Command{
	Use:   "nodes",
	Short: "Display the nodes of the cluster.",
	Long:  `Show what nodes are part of the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		nodes, err := v.GetNodes()

		if err != nil {
			fmt.Printf("Error getting nodes: %s\n", err)
			os.Exit(1)
		}

		header := []string{"Master", "Role", "Name", "Ip", "Id"}
		rows := [][]string{}
		for _, node := range nodes {
			row := []string{
				node.Master,
				node.Role,
				node.Name,
				node.Ip,
				node.Id,
			}

			rows = append(rows, row)
		}

		table := renderTable(rows, header)
		fmt.Println(table)
	},
}
