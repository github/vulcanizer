package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var nodesToGetHotThreads []string

func init() {
	cmdHotThreads.Flags().StringArrayVarP(&nodesToGetHotThreads, "nodes", "n", []string{}, "Elasticsearch nodes to get hot threads for. (optional, omitted means all nodes)")
	rootCmd.AddCommand(cmdHotThreads)
}

var cmdHotThreads = &cobra.Command{
	Use:   "hotthreads",
	Short: "Display the current hot threads by node in the cluster.",
	Long:  `Show the current hot threads across a set of nodes within the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		if len(nodesToGetHotThreads) == 0 {
			threads, err := v.GetHotThreads()
			if err != nil {
				fmt.Printf("Error getting hot threads: %s\n", err)
				os.Exit(1)
			}
			fmt.Println(threads)
			return
		}

		threads, err := v.GetNodesHotThreads(nodesToGetHotThreads)
		if err != nil {
			fmt.Printf("Error getting mappings: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(threads)
	},
}
