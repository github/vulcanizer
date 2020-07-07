package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

var shortTableOutput bool

func init() {
	cmdNodeAllocations.Flags().BoolVarP(&shortTableOutput, "short", "s", false, "Shorter, more compact table output")
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

		var header []string
		var rows [][]string
		if shortTableOutput {
			header = []string{"Role", "Name", "Avail", "Used", "Total", "%", "Indices", "Shards", "Ip"}
			rows = [][]string{}
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].Name < nodes[j].Name
			})
			for _, node := range nodes {
				row := []string{
					fmt.Sprintf("%s%s", node.Master, node.Role),
					node.Name,
					node.DiskAvail,
					node.DiskUsed,
					node.DiskTotal,
					node.DiskPercent,
					node.DiskIndices,
					node.Shards,
					node.Ip,
				}

				rows = append(rows, row)
			}
		} else {
			header = []string{"Master", "Role", "Name", "Disk Avail", "Disk Indices", "Disk Percent", "Disk Total", "Disk Used", "Shards", "Ip", "Id", "JDK", "Version"}
			rows = [][]string{}
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].Name < nodes[j].Name
			})
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
		}

		table := renderTable(rows, header)
		fmt.Println(table)
	},
}
