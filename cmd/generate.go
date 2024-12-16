package cmd

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-sync/generator"
	"log"
	"os"
	"text/template"
)

var generateCmd = &cobra.Command{
	Use:   "generate path/to/docker-compose.yml",
	Short: "Generate a lagoon-sync configuration stanza from a docker-compose file",
	Long: `Attempts to generate a lagoon-sync configuration from a docker-compose file.
Currently supports filesystem definitions, mariadb/mysql services, and postgres.
`,

	Args: cobra.MinimumNArgs(1),
	Run:  genCommandRun,
}

func genCommandRun(cmd *cobra.Command, args []string) {

	str, gerr := generator.RunWizard()

	if gerr == nil || gerr != nil {
		fmt.Println(str)
		return
	}
	_, err := os.Stat(args[0])
	if err != nil {
		log.Fatal(err)
	}

	project, err := generator.LoadComposeFile(args[0])
	if err != nil {
		log.Fatal(err)
	}

	services := generator.ProcessServicesFromCompose(project)

	stanza, err := generator.BuildConfigStanzaFromServices(services)

	const yamlTemplate = `
# Copy the following and add it to your .lagoon.yml file (see https://docs.lagoon.sh/concepts-basics/lagoon-yml/)

{{ .Sync }}
`

	tmpl, err := template.New("yaml").Parse(yamlTemplate)
	if err != nil {
		log.Fatal(err)
	}

	var output bytes.Buffer
	err = tmpl.Execute(&output, struct {
		Sync string
	}{
		Sync: stanza,
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output.String())
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
