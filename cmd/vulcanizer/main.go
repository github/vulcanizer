package main

import (
	"bytes"
	"fmt"
	"github.com/github/vulcanizer"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getConfiguration() (string, int, *vulcanizer.Auth) {
	var host string
	var port int
	var auth vulcanizer.Auth

	if viper.GetString("cluster") != "" {
		config := viper.Sub(viper.GetString("cluster"))

		if config == nil {
			fmt.Printf("Could not retrieve configuration for cluster \"%s\"\n", viper.GetString("cluster"))
			os.Exit(1)
		}

		host = config.GetString("host")
		port = config.GetInt("port")
		auth = vulcanizer.Auth{
			User:     config.GetString("user"),
			Password: config.GetString("password"),
		}
	} else {
		host = viper.GetString("host")
		port = viper.GetInt("port")
		auth = vulcanizer.Auth{
			User:     viper.GetString("user"),
			Password: viper.GetString("password"),
		}
	}

	return host, port, &auth
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
	rootCmd.PersistentFlags().StringP("user", "", "", "User to use during authentication")
	rootCmd.PersistentFlags().StringP("password", "", "", "Password to use during authentication")
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
	err = viper.BindPFlag("user", rootCmd.PersistentFlags().Lookup("user"))
	if err != nil {
		fmt.Printf("Error binding user flag: %s \n", err)
		os.Exit(1)
	}
	err = viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
	if err != nil {
		fmt.Printf("Error binding password flag: %s \n", err)
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
