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
	Long:  `Get detailed information about what consitutes the health of the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		health, err := v.GetHealth()

		if err != nil {
			fmt.Printf("Error getting cluster health: %s\n", err)
			os.Exit(1)
		}

		fmt.Println(health[0].Message)

		header := []string{"Cluster", "Status", "Relocating", "Initializing", "Unassigned", "Active %"}
		rows := [][]string{}
		for _, cluster := range health {
			row := []string{
				cluster.Cluster,
				cluster.Status,
				strconv.Itoa(cluster.RelocatingShards),
				strconv.Itoa(cluster.InitializingShards),
				strconv.Itoa(cluster.UnassignedShards),
				cluster.ActiveShardsPercentage,
			}

			rows = append(rows, row)
		}

		fmt.Println(renderTable(rows, header))
	},
}
