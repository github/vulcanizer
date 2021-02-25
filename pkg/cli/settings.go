package cli

import (
	"fmt"
	"os"

	"github.com/github/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	setupReloadSecureCommand()

	rootCmd.AddCommand(cmdSettings)
}

func setupReloadSecureCommand() {
	cmdSettingsReloadSecure.Flags().StringP("keystore_password", "", "", "Keystore password to reload the secure settings if enabled")
	cmdSettings.AddCommand(cmdSettingsReloadSecure)
}

func printSettings(settings []vulcanizer.Setting, name string) {
	if len(settings) == 0 {
		fmt.Printf("No %s are set.\n", name)
		return
	}

	header := []string{name, "Value"}
	rows := [][]string{}

	for _, setting := range settings {
		row := []string{
			setting.Setting,
			setting.Value,
		}

		rows = append(rows, row)
	}

	table := renderTable(rows, header)
	fmt.Println(table)
}

var cmdSettings = &cobra.Command{
	Use:   "settings",
	Short: "Display all the settings of the cluster.",
	Long:  `This command displays all the transient and persistent settings that have been set on the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		clusterSettings, err := v.GetClusterSettings()

		if err != nil {
			fmt.Printf("Error getting settings: %s\n", err)
			os.Exit(1)
		}

		printSettings(clusterSettings.PersistentSettings, "persistent settings")
		printSettings(clusterSettings.TransientSettings, "transient settings")
	},
}

var cmdSettingsReloadSecure = &cobra.Command{
	Use:   "reload",
	Short: "Reload the secure settings.",
	Long:  `This command calls the reload secure settings API on all nodes.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		password, err := cmd.Flags().GetString("keystore_password")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: keystore_password. Error: %s\n", err)
			os.Exit(1)
		}

		var reloadResponse vulcanizer.ReloadSecureSettingsResponse
		var reloadError error

		if password == "" {
			reloadResponse, reloadError = v.ReloadSecureSettings()
		} else {
			reloadResponse, reloadError = v.ReloadSecureSettingsWithPassword(password)
		}

		if reloadError != nil {
			fmt.Printf("Error reloading secure settings settings: %s\n", reloadError)
			os.Exit(1)
		}

		header := []string{"Node", "Reload status"}
		rows := [][]string{}

		for _, node := range reloadResponse.Nodes {
			row := []string{node.Name}

			if node.ReloadException == nil {
				row = append(row, "Successfully reloaded")
			} else {
				row = append(row, fmt.Sprintf("Exception type: %s, reason: %s", node.ReloadException.Type, node.ReloadException.Reason))
			}

			rows = append(rows, row)
		}

		table := renderTable(rows, header)
		fmt.Println(table)
	},
}
