package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getConfiguration() (string, int) {
	var host string
	var port int

	if viper.GetString("cluster") != "" {
		config := viper.Sub(viper.GetString("cluster"))

		if config == nil {
			fmt.Printf("Could not retrieve configuration for cluster \"%s\"\n", viper.GetString("cluster"))
			os.Exit(1)
		}

		host = config.GetString("host")
		port = config.GetInt("port")
	} else {
		host = viper.GetString("host")
		port = viper.GetInt("port")
	}

	return host, port
}

func renderTable(rows [][]string, header []string) string {
	var result bytes.Buffer
	table := tablewriter.NewWriter(&result)
	table.SetHeader(header)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(rows)
	table.Render()
	return result.String()
}

var rootCmd = &cobra.Command{Use: "vulcanizer"}

func main() {

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("host", "", "localhost", "Host to connect to")
	rootCmd.PersistentFlags().IntP("port", "p", 9200, "Port to connect to")
	rootCmd.PersistentFlags().StringP("cluster", "c", "", "Cluster to connect to defined in config file")
	rootCmd.PersistentFlags().StringP("configFile", "f", "", "Configuration file to read in (default to \"~/.vulcanizer.yaml\")")

	err := viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))
	if err != nil {
		fmt.Printf("Error binding host configuration flag: %s \n", err)
		os.Exit(1)
	}
	err = viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	if err != nil {
		fmt.Printf("Error binding port configuration flag: %s \n", err)
		os.Exit(1)
	}
	err = viper.BindPFlag("cluster", rootCmd.PersistentFlags().Lookup("cluster"))
	if err != nil {
		fmt.Printf("Error binding cluster configuration flag: %s \n", err)
		os.Exit(1)
	}

	err = rootCmd.Execute()
	if err != nil {
		fmt.Printf("Error executing root command: %s \n", err)
		os.Exit(1)
	}
}

func initConfig() {

	customCfgFile, err := rootCmd.Flags().GetString("configFile")
	if err != nil {
		fmt.Printf("Error reading in argument: configFile. Error: %s\n", err)
		os.Exit(1)
	}

	if customCfgFile != "" {
		viper.SetConfigFile(customCfgFile)
	} else {
		viper.AddConfigPath("$HOME")
		viper.SetConfigName(".vulcanizer")
	}

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Fatal error config file: %s \n", err)
		os.Exit(1)
	}
}
