package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cmdNodeHeap)
}

var cmdNodeHeap = &cobra.Command{
	Use:   "heap",
	Short: "Display the node heap stats.",
	Long:  `Show node heap stats and settings.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		nodeStats, err := v.GetNodeJVMStats()

		if err != nil {
			fmt.Printf("Error getting node JVM stats: %s\n", err)
			os.Exit(1)
		}

		var header []string
		var rows [][]string
		header = []string{"Name", "Role", "Heap Max", "Heap Used", "Heap %", "Non-Heap Committed", "Non-Heap Used"}
		rows = [][]string{}
		sort.Slice(nodeStats, func(i, j int) bool {
			return nodeStats[i].Name < nodeStats[j].Name
		})
		for _, node := range nodeStats {
			row := []string{
				node.Name,
				node.Role,
				byteCountSI(int64(node.JVMStats.HeapMaxBytes)),
				byteCountSI(int64(node.JVMStats.HeapUsedBytes)),
				fmt.Sprintf("%d %%", node.JVMStats.HeapUsedPercentage),
				byteCountSI(int64(node.JVMStats.NonHeapCommittedBytes)),
				byteCountSI(int64(node.JVMStats.NonHeapUsedBytes)),
			}

			rows = append(rows, row)
		}

		table := renderTable(rows, header)
		fmt.Println(table)
	},
}

func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
