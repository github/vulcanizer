package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var settingToUpdate, valueToUpdate string

var removeValue bool

func init() {

	cmdSettingUpdate.Flags().StringVarP(&settingToUpdate, "setting", "s", "", "Elasticsearch cluster setting to update (required)")
	err := cmdSettingUpdate.MarkFlagRequired("setting")
	if err != nil {
		fmt.Printf("Error binding setting configuration flag: %s \n", err)
		os.Exit(1)
	}

	cmdSettingUpdate.Flags().StringVarP(&valueToUpdate, "value", "v", "", "Value of the Elasticsearch cluster setting to update (can't be used with \"--remove\")")

	cmdSettingUpdate.Flags().BoolVar(&removeValue, "remove", false, "Remove provided cluster setting, resetting it to default configuration (can't be used with \"--value|-v\")")

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

		if cmd.Flags().Changed("value") && cmd.Flags().Changed("remove") {
			fmt.Println("Can't set both \"--value|-v\" and \"--remove\" options")
			fmt.Print(cmd.UsageString())
			os.Exit(1)
		}
		if !cmd.Flags().Changed("value") && !cmd.Flags().Changed("remove") {
			fmt.Println("Error: requires one of \"--value|-v\" or \"--remove\"")
			fmt.Print(cmd.UsageString())
			os.Exit(1)
		}
		v := getClient()

		var ptrValueToUpdate *string

		if removeValue {
			ptrValueToUpdate = nil
		} else {
			ptrValueToUpdate = &valueToUpdate
		}

		existingValue, newValue, err := v.SetClusterSetting(settingToUpdate, ptrValueToUpdate)

		if err != nil {
			fmt.Printf("Error when trying to update \"%s\" to \"%s\"\n", settingToUpdate, printableNil(ptrValueToUpdate))
			fmt.Printf("Error is: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Updated setting %s\n", settingToUpdate)
		fmt.Printf("\tOld value: %s\n", printableNil(existingValue))
		fmt.Printf("\tNew value: %s\n", printableNil(newValue))
	},
}
