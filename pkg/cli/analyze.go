package cli

import (
	"fmt"
	"os"

	"github.com/leosunmo/vulcanizer"
	"github.com/spf13/cobra"
)

func init() {
	cmdAnalyze.Flags().StringP("analyzer", "a", "", "The analyzer used to analyze the text Overrides field and index settings.")

	cmdAnalyze.Flags().StringP("field", "", "", "Use analyzer settings of the given field to analyze against. Overridden by analyzer setting. index setting also required.")

	cmdAnalyze.Flags().StringP("index", "i", "", "Specifies which index to look for field setting. Overridden by analyzer setting. field setting also required.")

	cmdAnalyze.Flags().StringP("text", "t", "", "Text to analyze (required).")
	err := cmdAnalyze.MarkFlagRequired("text")
	if err != nil {
		fmt.Printf("Error binding text configuration flag: %s \n", err)
		os.Exit(1)
	}

	rootCmd.AddCommand(cmdAnalyze)
}

var cmdAnalyze = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze text given an analyzer or a field and index.",
	Long:  `Use Elasticsearch's analyze API to display the tokens of example text. Either "analyzer" OR "field" and "index" arguments are required.`,
	Run: func(cmd *cobra.Command, args []string) {

		v := getClient()

		text, err := cmd.Flags().GetString("text")
		if err != nil {
			fmt.Printf("Could not retrieve required argument: text. Error: %s\n", err)
			os.Exit(1)
		}

		analyzer, _ := cmd.Flags().GetString("analyzer")

		var tokens []vulcanizer.Token

		if analyzer != "" {
			ts, err := v.AnalyzeText(analyzer, text)
			if err != nil {
				fmt.Printf("Error calling analyze API: %s\n", err)
				os.Exit(1)
			}

			tokens = ts
		} else {
			field, _ := cmd.Flags().GetString("field")
			index, _ := cmd.Flags().GetString("index")

			if field == "" || index == "" {
				fmt.Println("Either `analyzer` or `field` and `index` arguments are required.")
				os.Exit(1)
			}

			ts, err := v.AnalyzeTextWithField(index, field, text)
			if err != nil {
				fmt.Printf("Error calling analyze API: %s\n", err)
				os.Exit(1)
			}

			tokens = ts
		}

		fmt.Printf("Printing tokens for the provided text: %s\n", text)
		for _, t := range tokens {
			fmt.Printf("\t* %s - %s\n", t.Text, t.Type)
		}
	},
}
