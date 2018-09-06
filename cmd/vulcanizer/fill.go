package main

import (
	"fmt"

	v "github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

var serverToFill string

func init() {
	cmdFillServer.Flags().StringVarP(&serverToFill, "name", "n", "", "Elasticsearch node name to fill (required)")
	err := cmdFillServer.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}

	cmdFill.AddCommand(cmdFillServer, cmdFillAll)
	rootCmd.AddCommand(cmdFill)
}

var cmdFill = &cobra.Command{
	Use:   "fill",
	Short: "Fill servers with data, removing shard allocation exclusion rules.",
	Long:  `Use the subcommands to remove shard allocation exclusion rules from one server or all servers.`,
}

var cmdFillAll = &cobra.Command{
	Use:   "all",
	Short: "Fill all servers with data, removing all exclusion rules.",
	Long:  `This command will remove all shard allocation exclusion rules from the cluster, allowing all servers to fill with data.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()

		excludeSettings := v.FillAll(host, port)

		fmt.Printf("Current allocation exclude settings: %+v\n", excludeSettings)
	},
}

var cmdFillServer = &cobra.Command{
	Use:   "server",
	Short: "Fill one server with data, removing exclusion rules from it.",
	Long:  `This command will remove shard allocation exclusion rules from a particular Elasticsearch node, allowing shards to allocated to it.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()

		serverFilling, excludedServers := v.FillOneServer(host, port, serverToFill)

		fmt.Printf("Server \"%s\" removed from allocation rules.\n", serverFilling)
		fmt.Printf("Servers \"%s\" are still being excluded from allocation.\n", excludedServers)
	},
}
