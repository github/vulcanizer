package main

import (
	"fmt"

	v "github.com/github/vulcanizer"
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
		fmt.Printf("config host: %s, port: %v\n", host, port)

		caption, rows, headers := v.GetHealth(host, port)

		fmt.Println(caption)
		fmt.Println(renderTable(rows, headers))
	},
}
