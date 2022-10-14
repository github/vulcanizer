package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var indexToGetMappings string

func init() {
	cmdIndexMappings.Flags().StringVarP(&indexToGetMappings, "index", "i", "", "Elasticsearch index to retrieve mappings from (required)")
	err := cmdIndexMappings.MarkFlagRequired("index")
	if err != nil {
		fmt.Printf("Error binding name configuration flag: %s \n", err)
		os.Exit(1)
	}
	rootCmd.AddCommand(cmdIndexMappings)
}

var cmdIndexMappings = &cobra.Command{
	Use:   "mappings",
	Short: "Display the mappings of the specified index.",
	Long:  `Show the mappings of the specified index within the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		mappings, err := v.GetPrettyIndexMappings(indexToGetMappings)

		if err != nil {
			fmt.Printf("Error getting mappings: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(mappings)
	},
}
