package main

import (
	"fmt"
	"os"

	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	setupAliasesListSubCommand()
	rootCmd.AddCommand(cmdAliases)
}

func setupAliasesListSubCommand() {
	cmdAliases.AddCommand(cmdAliasesList)
}

var cmdAliases = &cobra.Command{
	Use:   "aliases",
	Short: "Interact with aliases of the cluster.",
	Long:  `Use the list subcommand.`,
}

var cmdAliasesList = &cobra.Command{
	Use:   "list",
	Short: "Display the aliases of the cluster",
	Long:  `Show what aliases are created on the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port, auth := getConfiguration()
		v := vulcanizer.NewClient(host, port)
		v.Auth = auth
		aliases, err := v.GetAliases()

		if err != nil {
			fmt.Printf("Error getting aliases: %s\n", err)
			os.Exit(1)
		}

		header := []string{"Alias", "Index", "Filter", "routing.index", "routing.search"}
		rows := [][]string{}

		for _, alias := range aliases {
			row := []string{
				alias.Name,
				alias.IndexName,
				alias.Filter,
				alias.RoutingIndex,
				alias.RoutingSearch,
			}

			rows = append(rows, row)
		}

		table := renderTable(rows, header)
		fmt.Println(table)
	},
}
