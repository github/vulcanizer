package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/github/vulcanizer"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Host          string
	Port          int
	Protocol      string
	Path          string
	User          string
	Password      string
	TLSSkipVerify bool
}

func getConfiguration() Config {
	var config Config

	v := viper.GetViper()

	if viper.GetString("cluster") != "" {
		v = viper.Sub(viper.GetString("cluster"))

		if v == nil {
			fmt.Printf("Could not retrieve configuration for cluster \"%s\"\n", viper.GetString("cluster"))
			os.Exit(1)
		}
	}

	config = Config{
		Host:     v.GetString("host"),
		Port:     v.GetInt("port"),
		Protocol: v.GetString("protocol"),
		Path:     v.GetString("path"),

		User:     v.GetString("user"),
		Password: v.GetString("password"),

		TLSSkipVerify: v.GetBool("skipverify"),
	}
	return config
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
	rootCmd.PersistentFlags().StringP("path", "", "/", "Path to prepend to queries, in case Elasticsearch is behind a reverse proxy")
	rootCmd.PersistentFlags().StringP("protocol", "", "http", "Protocol to use when querying the cluster. Either 'http' or 'https'. Defaults to 'http'")
	rootCmd.PersistentFlags().StringP("skipverify", "k", "false", "Skip verifying server's TLS certificate. Defaults to 'false', ie. verify the server's certificate")
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
	err = viper.BindPFlag("path", rootCmd.PersistentFlags().Lookup("path"))
	if err != nil {
		fmt.Printf("Error binding cluster configuration flag: %s \n", err)
		os.Exit(1)
	}
	err = viper.BindPFlag("protocol", rootCmd.PersistentFlags().Lookup("protocol"))
	if err != nil {
		fmt.Printf("Error binding cluster configuration flag: %s \n", err)
		os.Exit(1)
	}
	err = viper.BindPFlag("skipverify", rootCmd.PersistentFlags().Lookup("skipverify"))
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
