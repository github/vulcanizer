package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cmdNodes)
}

var cmdNodes = &cobra.Command{
	Use:   "nodes",
	Short: "Display the nodes of the cluster.",
	Long:  `Show what nodes are part of the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		nodes, err := v.GetNodes()

		if err != nil {
			fmt.Printf("Error getting nodes: %s\n", err)
			os.Exit(1)
		}

		header := []string{"Master", "Role", "Name", "Ip", "Id", "JDK", "Version"}
		rows := [][]string{}
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Name < nodes[j].Name
		})
		for _, node := range nodes {
			row := []string{
				node.Master,
				node.Role,
				node.Name,
				node.IP,
				node.ID,
				node.Jdk,
				node.Version,
			}

			rows = append(rows, row)
		}

		table := renderTable(rows, header)
		fmt.Println(table)
	},
}
