package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cmdHealth)
}

var cmdHealth = &cobra.Command{
	Use:   "health",
	Short: "Display the health of the cluster.",
	Long:  `Get detailed information about what constitutes the health of the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		health, err := v.GetHealth()

		if err != nil {
			fmt.Printf("Error getting cluster health: %s\n", err)
			os.Exit(1)
		}

		fmt.Println(health.Message)

		header := []string{"Cluster", "Status", "Relocating", "Initializing", "Unassigned", "Active %"}
		rows := [][]string{}
		row := []string{
			health.Cluster,
			health.Status,
			strconv.Itoa(health.RelocatingShards),
			strconv.Itoa(health.InitializingShards),
			strconv.Itoa(health.UnassignedShards),
			strconv.FormatFloat(health.ActiveShardsPercentage, 'f', -1, 32),
		}
		rows = append(rows, row)

		fmt.Println(renderTable(rows, header))
	},
}
