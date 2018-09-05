package main

import (
	"bytes"
	"fmt"

	"github.com/olekukonko/tablewriter"
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

func renderTable(rows [][]string, header []string) string {
	var result bytes.Buffer
	table := tablewriter.NewWriter(&result)
	table.SetHeader(header)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.AppendBulk(rows)
	table.Render()
	return result.String()
}

var rootCmd = &cobra.Command{Use: "app"}

func main() {

	viper.SetConfigName(".vulcanizer")
	viper.AddConfigPath("$HOME")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	rootCmd.PersistentFlags().StringP("host", "", "", "Host to connect to")
	rootCmd.PersistentFlags().IntP("port", "p", 9200, "Port to connect to")
	rootCmd.PersistentFlags().StringP("cluster", "c", "", "Cluster to connect to defined in config file")

	viper.BindPFlag("host", rootCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("cluster", rootCmd.PersistentFlags().Lookup("cluster"))

	rootCmd.Execute()
}
