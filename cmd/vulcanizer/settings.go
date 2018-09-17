package main

import (
	"fmt"
	"os"

	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cmdSettings)
}

func printSettings(settings []vulcanizer.ClusterSetting, name string) {
	if len(settings) == 0 {
		fmt.Println(fmt.Sprintf("No %s are set.\n", name))
		return
	}

	header := []string{name, "Value"}
	rows := [][]string{}

	for _, setting := range settings {
		row := []string{
			setting.Setting,
			setting.Value,
		}

		rows = append(rows, row)
	}

	table := renderTable(rows, header)
	fmt.Println(table)
}

var cmdSettings = &cobra.Command{
	Use:   "settings",
	Short: "Display all the settings of the cluster.",
	Long:  `This command displays all the transient and persisent settings that have been set on the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		clusterSettings, err := v.GetSettings()

		if err != nil {
			fmt.Printf("Error getting settings: %s\n", err)
			os.Exit(1)
		}

		printSettings(clusterSettings.PersistentSettings, "persistent settings")
		printSettings(clusterSettings.TransientSettings, "transient settings")
	},
}
