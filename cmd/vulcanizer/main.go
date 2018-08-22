package main

import (
	"fmt"

	v "github.com/github/vulcanizer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getConfiguration() (string, int) {
	var host string
	var port int

	if viper.GetString("cluster") != "" {
		config := viper.Sub(viper.GetString("cluster"))
		host = config.GetString("host")
		port = config.GetInt("port")
	} else {
		host = viper.GetString("host")
		port = viper.GetInt("port")
	}

	return host, port
}

func main() {

	viper.SetConfigName(".vulcanizer")
	viper.AddConfigPath("$HOME")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	var cmdHealth = &cobra.Command{
		Use:   "health",
		Short: "Display the health of the cluster.",
		Long:  `Get detailed information about what consitutes the health of the cluster`,
		Run: func(cmd *cobra.Command, args []string) {

			host, port := getConfiguration()
			fmt.Printf("viper config host: %s, port: %v\n", host, port)

			caption, values, _ := v.GetHealth(host, port)

			fmt.Println(caption)
			fmt.Println(values)
		},
	}

	var cmdIndices = &cobra.Command{
		Use:   "indices",
		Short: "Display the indices of the cluster.",
		Long:  `Show what indices are created on the give cluster.`,
		Run: func(cmd *cobra.Command, args []string) {

			host, port := getConfiguration()
			fmt.Printf("viper config host: %s, port: %v\n", host, port)
		},
	}

	var cmdNodes = &cobra.Command{
		Use:   "nodes",
		Short: "Display the nodes of the cluster.",
		Long:  `Show what nodes are part of the cluster.`,
		Run: func(cmd *cobra.Command, args []string) {
			host, port := getConfiguration()
			fmt.Printf("viper config host: %s, port: %v\n", host, port)
		},
	}

	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(cmdHealth, cmdIndices, cmdNodes)

	rootCmd.PersistentFlags().StringP("host", "u", "", "Host to connect to")
	rootCmd.PersistentFlags().IntP("port", "p", 0, "Port to connect to")
	rootCmd.PersistentFlags().StringP("cluster", "c", "", "Cluster to connect to defined in config file")

	viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("cluster", rootCmd.PersistentFlags().Lookup("cluster"))

	rootCmd.Execute()
}
