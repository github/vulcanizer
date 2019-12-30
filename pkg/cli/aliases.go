package cli

import (
	"fmt"
	"os"

	"github.com/leosunmo/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	setupAliasesListSubCommand()
	setupAliasesUpdateSubCommand()
	setupAliasesAddSubCommand()
	setupAliasesDeleteSubCommand()
	rootCmd.AddCommand(cmdAliases)
}

func setupAliasesListSubCommand() {
	cmdAliases.AddCommand(cmdAliasesList)
}

func setupAliasesUpdateSubCommand() {
	cmdAliasesUpdate.Flags().StringP("index", "i", "", "Index from which to delete the alias (required)")
	err := cmdAliasesUpdate.MarkFlagRequired("index")
	if err != nil {
		fmt.Printf("Error binding index flag: %s \n", err)
		os.Exit(1)
	}

	cmdAliasesUpdate.Flags().StringP("old-alias", "", "", "Alias to be deleted from the specified index (required)")
	err = cmdAliasesUpdate.MarkFlagRequired("old-alias")
	if err != nil {
		fmt.Printf("Error binding old-alias flag: %s \n", err)
		os.Exit(1)
	}

	cmdAliasesUpdate.Flags().StringP("new-alias", "", "", "Alias to be added to the specified index (required)")
	err = cmdAliasesUpdate.MarkFlagRequired("new-alias")
	if err != nil {
		fmt.Printf("Error binding new-alias flag: %s \n", err)
		os.Exit(1)
	}

	cmdAliases.AddCommand(cmdAliasesUpdate)
}

func setupAliasesAddSubCommand() {
	cmdAliasesAdd.Flags().StringP("index", "i", "", "Index to which the alias should be added (required)")
	err := cmdAliasesAdd.MarkFlagRequired("index")
	if err != nil {
		fmt.Printf("Error binding index flag: %s \n", err)
		os.Exit(1)
	}

	cmdAliasesAdd.Flags().StringP("alias", "a", "", "Alias to be added from the specified index (required)")
	err = cmdAliasesAdd.MarkFlagRequired("alias")
	if err != nil {
		fmt.Printf("Error binding alias flag: %s \n", err)
		os.Exit(1)
	}

	cmdAliases.AddCommand(cmdAliasesAdd)
}

func setupAliasesDeleteSubCommand() {
	cmdAliasesDelete.Flags().StringP("index", "i", "", "Index from which the alias should be deleted (required)")
	err := cmdAliasesDelete.MarkFlagRequired("index")
	if err != nil {
		fmt.Printf("Error binding index flag: %s \n", err)
		os.Exit(1)
	}

	cmdAliasesDelete.Flags().StringP("alias", "a", "", "Alias to be deleted to the specified index (required)")
	err = cmdAliasesDelete.MarkFlagRequired("alias")
	if err != nil {
		fmt.Printf("Error binding alias flag: %s \n", err)
		os.Exit(1)
	}

	cmdAliases.AddCommand(cmdAliasesDelete)
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
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		v := getClient()

		var err error
		var aliases []vulcanizer.Alias
		if len(args) > 0 {
			aliases, err = v.GetAliases(args[0])
		} else {
			aliases, err = v.GetAllAliases()
		}

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

var cmdAliasesUpdate = &cobra.Command{
	Use:   "update",
	Short: "Update an index's alias",
	Long:  `In one atomic operation delete an existing alias and create a new one for a given index on the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		indexName, err := cmd.Flags().GetString("index")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: index. Error: %s\n", err)
			os.Exit(1)
		}

		newAlias, err := cmd.Flags().GetString("new-alias")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: new-alias. Error: %s\n", err)
			os.Exit(1)
		}

		oldAlias, err := cmd.Flags().GetString("old-alias")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: old-alias. Error: %s\n", err)
			os.Exit(1)
		}

		actions := []vulcanizer.AliasAction{
			{
				ActionType: vulcanizer.AddAlias,
				IndexName:  indexName,
				AliasName:  newAlias,
			},
			{
				ActionType: vulcanizer.RemoveAlias,
				IndexName:  indexName,
				AliasName:  oldAlias,
			},
		}

		err = v.ModifyAliases(actions)
		if err != nil {
			fmt.Printf("Error while taking snapshot. Error: %s\n", err)
			os.Exit(1)
		}
	},
}

var cmdAliasesAdd = &cobra.Command{
	Use:   "add",
	Short: "Add an alias",
	Long:  `Add a new alias to a given index in the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		indexName, err := cmd.Flags().GetString("index")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: index. Error: %s\n", err)
			os.Exit(1)
		}

		aliasName, err := cmd.Flags().GetString("alias")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: alias. Error: %s\n", err)
			os.Exit(1)
		}

		actions := []vulcanizer.AliasAction{
			{
				ActionType: vulcanizer.AddAlias,
				IndexName:  indexName,
				AliasName:  aliasName,
			},
		}

		err = v.ModifyAliases(actions)
		if err != nil {
			fmt.Printf("Error while taking adding a new alias. Error: %s\n", err)
			os.Exit(1)
		}
	},
}

var cmdAliasesDelete = &cobra.Command{
	Use:   "delete",
	Short: "Delete an alias",
	Long:  `Delete an alias associated to a given index in the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		indexName, err := cmd.Flags().GetString("index")
		if err != nil || indexName == "" {
			fmt.Printf("Could not retrieve required argument: index. Error: %s\n", err)
			os.Exit(1)
		}

		aliasName, err := cmd.Flags().GetString("alias")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: alias. Error: %s\n", err)
			os.Exit(1)
		}

		actions := []vulcanizer.AliasAction{
			{
				ActionType: vulcanizer.RemoveAlias,
				IndexName:  indexName,
				AliasName:  aliasName,
			},
		}

		err = v.ModifyAliases(actions)
		if err != nil {
			fmt.Printf("Error while deleting the alias. Error: %s\n", err)
			os.Exit(1)
		}
	},
}
