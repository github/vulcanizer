package main

import (
	"fmt"

	v "github.com/github/vulcanizer"
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
		fmt.Printf("config host: %s, port: %v\n", host, port)
		rows, header := v.GetNodes(host, port)

		table := renderTable(rows, header)

		fmt.Println(table)
	},
}
