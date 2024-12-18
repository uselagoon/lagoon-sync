package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-sync/generator"
	"log"
	"os"
)

var wizardCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Generate a lagoon-sync configuration stanza interactively",
	Long:  ``,
	Run:   genwizCommandRun,
}

func genwizCommandRun(cmd *cobra.Command, args []string) {

	str, gerr := generator.RunWizard()

	if gerr != nil {
		log.Fatal(gerr)
	}

	if outputfile != "" {
		// Create or open the file
		file, err := os.Create(outputfile)
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer file.Close()

		// Write the string to the file
		_, err = file.WriteString(str)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
		fmt.Println("Successfully wrote output to : " + outputfile)
	} else {
		fmt.Println(str)
	}

}

func init() {
	rootCmd.AddCommand(wizardCmd)
	wizardCmd.PersistentFlags().StringVarP(&outputfile, "outputfile", "o", "", "Write output to file - outputs to STDOUT if unset")
}
