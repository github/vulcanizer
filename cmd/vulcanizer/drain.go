package main

import (
	"fmt"
	"os"

	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

var serverToDrain string

func init() {
	cmdDrainServer.Flags().StringVarP(&serverToDrain, "name", "n", "", "Elasticsearch node name to drain (required)")
	err := cmdDrainServer.MarkFlagRequired("name")
	if err != nil {
		fmt.Printf("Error binding name configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdDrain.AddCommand(cmdDrainServer, cmdDrainStatus)
	rootCmd.AddCommand(cmdDrain)
}

var cmdDrain = &cobra.Command{
	Use:   "drain",
	Short: "Drain a server or see what servers are draining.",
	Long:  `Use the subcommands to drain a server or to see what servers are currently draining.`,
}

var cmdDrainServer = &cobra.Command{
	Use:   "server",
	Short: "Drain a server by excluding shards from it.",
	Long:  `This command will set the shard allocation rules to exclude the given server name. This will cause shards to be moved away from this server, draining the data away.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		fmt.Printf("drain server name is: %s\n", serverToDrain)

		excludedServers, err := v.DrainServer(serverToDrain)
		if err != nil {
			fmt.Printf("Error getting exclude settings: %s \n", err)
			os.Exit(1)
		}

		fmt.Printf("draining servers: %+v\n", excludedServers)
	},
}

var cmdDrainStatus = &cobra.Command{
	Use:   "status",
	Short: "See what servers are set to drain.",
	Long:  `This command will display what servers are set in the clusters allocation exclude rules.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		excludeSettings, err := v.GetClusterExcludeSettings()
		if err != nil {
			fmt.Printf("Error getting exclude settings: %s \n", err)
			os.Exit(1)
		}
		fmt.Printf("drain status: %+v\n", excludeSettings)
	},
}
