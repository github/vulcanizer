package main

import (
	"fmt"

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
		caption, rows, headers := v.GetHealth()

		fmt.Println(caption)
		fmt.Println(renderTable(rows, headers))
	},
}
