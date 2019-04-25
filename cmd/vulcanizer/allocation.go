package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	cmdAllocation.AddCommand(cmdAllocationEnable, cmdAllocationDisable)
	rootCmd.AddCommand(cmdAllocation)
}

var cmdAllocation = &cobra.Command{
	Use:   "allocation",
	Short: "Set shard allocation on the cluster.",
	Long:  `This command sets allocation on the cluster. It is accessed by the enable/disable subcommands.`,
}

var cmdAllocationEnable = &cobra.Command{
	Use:   "enable",
	Short: "Enable allocation on the cluster.",
	Long:  `This commands enables allocation on the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		response, err := v.SetAllocation("enable")
		if err != nil {
			fmt.Printf("Error setting allocation: %s \n", err)
			os.Exit(1)
		}
		fmt.Printf("Enabling allocation:\n")
		fmt.Printf("Allocation set to %s\n", response)
	},
}

var cmdAllocationDisable = &cobra.Command{
	Use:   "disable",
	Short: "Disable allocation on the cluster.",
	Long:  `This commands disables allocation on the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		response, err := v.SetAllocation("disable")
		if err != nil {
			fmt.Printf("Error setting allocation: %s \n", err)
			os.Exit(1)
		}
		fmt.Printf("Disabling allocation:\n")
		fmt.Printf("Allocation set to %s\n", response)
	},
}
