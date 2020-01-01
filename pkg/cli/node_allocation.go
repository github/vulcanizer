package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cmdNodeAllocations)
}

var cmdNodeAllocations = &cobra.Command{
	Use:   "nodeallocations",
	Short: "Display the nodes of the cluster and their disk usage/allocation.",
	Long:  `Show disk allocation of the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		nodes, err := v.GetNodeAllocations()

		if err != nil {
			fmt.Printf("Error getting nodes: %s\n", err)
			os.Exit(1)
		}

		header := []string{"Master", "Role", "Name", "Disk Avail", "Disk Indices", "Disk Percent", "Disk Total", "Disk Used", "Shards", "Ip", "Id", "JDK", "Version"}
		rows := [][]string{}
		for _, node := range nodes {
			row := []string{
				node.Master,
				node.Role,
				node.Name,
				node.DiskAvail,
				node.DiskIndices,
				node.DiskPercent,
				node.DiskTotal,
				node.DiskUsed,
				node.Shards,
				node.Ip,
				node.Id,
				node.Jdk,
				node.Version,
			}

			rows = append(rows, row)
		}

		table := renderTable(rows, header)
		fmt.Println(table)
	},
}
