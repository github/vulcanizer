package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var serverToFill string

func init() {
	cmdFillServer.Flags().StringVarP(&serverToFill, "name", "n", "", "Elasticsearch node name to fill (required)")
	err := cmdFillServer.MarkFlagRequired("name")
	if err != nil {
		fmt.Printf("Error binding name configuration flag: %s \n", err)
		os.Exit(1)
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

		v := getClient()

		excludeSettings, err := v.FillAll()
		if err != nil {
			fmt.Printf("Error calling Elasticsearch: %s \n", err)
			os.Exit(1)
		}

		fmt.Printf("Current allocation exclude settings: %+v\n", excludeSettings)
	},
}

var cmdFillServer = &cobra.Command{
	Use:   "server",
	Short: "Fill one server with data, removing exclusion rules from it.",
	Long:  `This command will remove shard allocation exclusion rules from a particular Elasticsearch node, allowing shards to allocated to it.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		excludeSettings, err := v.FillOneServer(serverToFill)
		if err != nil {
			fmt.Printf("Error calling Elasticsearch: %s \n", err)
			os.Exit(1)
		}

		fmt.Printf("Server \"%s\" removed from allocation rules.\n", serverToFill)
		fmt.Printf("Current exclude settings: %+v\n", excludeSettings)
	},
}
