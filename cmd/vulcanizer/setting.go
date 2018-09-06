package main

import (
	"fmt"
	"os"

	v "github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

var settingToUpdate, valueToUpdate string

func init() {

	cmdSettingUpdate.Flags().StringVarP(&settingToUpdate, "setting", "s", "", "Elasticsearch cluster setting to update (required)")
	err := cmdSettingUpdate.MarkFlagRequired("setting")
	if err != nil {
		panic(err)
	}

	cmdSettingUpdate.Flags().StringVarP(&valueToUpdate, "value", "v", "", "Value of the Elasticsearch cluster setting to update (required)")
	err = cmdSettingUpdate.MarkFlagRequired("value")
	if err != nil {
		panic(err)
	}

	cmdSetting.AddCommand(cmdSettingUpdate)
	rootCmd.AddCommand(cmdSetting)
}

var cmdSetting = &cobra.Command{
	Use:   "setting",
	Short: "Interact with cluster settings.",
	Long:  `Use the subcommands to update cluster settings.`,
}

var cmdSettingUpdate = &cobra.Command{
	Use:   "update",
	Short: "Update a cluster setting.",
	Long:  `This command will update the cluster's settings with the provided value.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()

		existingValue, newValue, err := v.SetSetting(host, port, settingToUpdate, valueToUpdate)

		if err != nil {
			fmt.Printf("Error when trying to update \"%s\" to \"%s\n", settingToUpdate, valueToUpdate)
			fmt.Printf("Error is: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Updated setting %s\n", settingToUpdate)
		fmt.Printf("\tOld value: %s\n", existingValue)
		fmt.Printf("\tNew value: %s\n", newValue)
	},
}
