package main

import (
	"fmt"

	vulcan "github.com/github/vulcan-go-opensource-lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {

	viper.SetConfigName(".vulcan")
	viper.AddConfigPath("$HOME")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	var cmdHealth = &cobra.Command{
		Use:   "health <cluster>",
		Short: "Display the health of the cluster.",
		Long:  `Get detailed information about what consitutes the health of the cluster`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			host := viper.GetString(fmt.Sprintf("%s.host", args[0]))
			port := viper.GetInt(fmt.Sprintf("%s.port", args[0]))

			fmt.Printf("viper config host: %s, port: %v\n", host, port)

			caption, values, _ := vulcan.GetHealth(host, port)

			fmt.Println(caption)
			fmt.Println(values)
		},
	}

	var cmdIndices = &cobra.Command{
		Use:   "indices <cluster>",
		Short: "Display the indices of the cluster.",
		Long:  `Show what indices are created on the give cluster.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Get indices on cluster", args[0])
		},
	}

	var cmdNodes = &cobra.Command{
		Use:   "nodes <cluster>",
		Short: "Display the nodes of the cluster.",
		Long:  `Show what nodes are part of the cluster.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Get nodes on cluster", args[0])
		},
	}

	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(cmdHealth, cmdIndices, cmdNodes)
	rootCmd.Execute()
}
