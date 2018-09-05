package main

import (
	"fmt"

	v "github.com/github/vulcanizer"
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
		host, port := getConfiguration()
		response := v.SetAllocation(host, port, "enable")
		fmt.Printf("Enabling allocation:\n")
		fmt.Printf("Allocation set to %s\n", response)
	},
}

var cmdAllocationDisable = &cobra.Command{
	Use:   "disable",
	Short: "Disable allocation on the cluster.",
	Long:  `This commands disables allocation on the given cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		host, port := getConfiguration()
		response := v.SetAllocation(host, port, "disable")
		fmt.Printf("Disabling allocation:\n")
		fmt.Printf("Allocation set to %s\n", response)
	},
}
