package generator

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/uselagoon/lagoon-sync/synchers"
)

func displayConfigTemplateData(d configTemplateData) {

	fmt.Println("\n\n--- Current Service List ---")
	if len(d.Mariadb) > 0 {
		fmt.Println("\tMariadb services:")
		for _, s := range d.Mariadb {
			fmt.Println("\t\t" + s.ServiceName)
		}
	}
	if len(d.Postgres) > 0 {
		fmt.Println("\tPostgres services:")
		for _, s := range d.Postgres {
			fmt.Println("\t\t" + s.ServiceName)
		}
	}
	if len(d.Filesystem) > 0 {
		fmt.Println("\tFilesystems to sync:")
		for _, s := range d.Filesystem {
			fmt.Println("\t\t" + s.ServiceName + ":" + s.Config.SyncPath)
		}
	}

	fmt.Println("\n")
}

func RunWizard() (string, error) {

	template := configTemplateData{}

	done := false
	const addMariadbString = "Add Mariadb"
	const addPostgressString = "Add Postgres"
	const addFSString = "Add filesystem"
	const exitString = "Exit"

	for !done {
		displayConfigTemplateData(template)
		prompt := &survey.Select{
			Message: "Choose a color:",
			Options: []string{addMariadbString, addPostgressString, addFSString, exitString},
		}
		var opt string
		// or define a default for the single call to `AskOne`
		// the answer will get written to the color variable
		survey.AskOne(prompt, &opt, survey.WithValidator(survey.Required))

		switch opt {
		case addMariadbString:
			addMariadbService(&template)
		case addPostgressString:
			addPostgresqlService(&template)
		case addFSString:
			addFilesystemSyncer(&template)
		case exitString:
			done = true

		}

		if opt == exitString {
			done = true
		}
	}

	return generateSyncStanza(template)
}

func addMariadbService(c *configTemplateData) {
	fmt.Println("\nAdding a mariadb service:")
	var qs = []*survey.Question{
		{
			Name:      "Servicename",
			Prompt:    &survey.Input{Message: "What is the name of the service (typically the service name in your docker file)?"},
			Validate:  survey.Required,
			Transform: survey.ToLower,
		},
	}
	answers := struct {
		Servicename string
	}{}
	err := survey.Ask(qs, &answers)
	if err != nil {
		fmt.Println(err)
		return
	}

	service, err := GenerateMariadbSyncRootFromService(LagoonServiceDefinition{
		ServiceName: answers.Servicename,
		ServiceType: synchers.MariadbSyncPlugin{}.GetPluginId(),
	})
	c.Mariadb = append(c.Mariadb, service)

}

func addPostgresqlService(c *configTemplateData) {
	fmt.Println("\nAdding a postgresql service:")
	var qs = []*survey.Question{
		{
			Name:      "Servicename",
			Prompt:    &survey.Input{Message: "What is the name of the service (typically the service name in your docker file)?"},
			Validate:  survey.Required,
			Transform: survey.ToLower,
		},
	}
	answers := struct {
		Servicename string
	}{}
	err := survey.Ask(qs, &answers)
	if err != nil {
		fmt.Println(err)
		return
	}

	service, err := GeneratePgqlSyncRootFromService(LagoonServiceDefinition{
		ServiceName: answers.Servicename,
		ServiceType: synchers.PostgresSyncPlugin{}.GetPluginId(),
	})
	c.Postgres = append(c.Postgres, service)
}

func addFilesystemSyncer(c *configTemplateData) {
	fmt.Println("\nAdding a File sync:")
	var qs = []*survey.Question{
		{
			Name:      "Servicename",
			Prompt:    &survey.Input{Message: "What is the name of the sync you'd like to setup (a useful name to refer to the sync, such as 'publicfiles')?"},
			Validate:  survey.Required,
			Transform: survey.ToLower,
		},
		{
			Name:      "Path",
			Prompt:    &survey.Input{Message: "What is the path you'd like to sync (eg. '/app/web/sites/default/files')?"},
			Validate:  survey.Required,
			Transform: survey.ToLower,
		},
	}
	answers := struct {
		Servicename string
		Path        string
	}{}
	err := survey.Ask(qs, &answers)
	if err != nil {
		fmt.Println(err)
		return
	}

	service, err := GenerateFilesSyncRootsFromServiceDefinition(LagoonServiceDefinition{
		ServiceName: answers.Servicename,
		ServiceType: synchers.FilesSyncPlugin{}.GetPluginId(),
		Labels: map[string]string{
			"lagoon.volumes." + answers.Servicename + ".path": answers.Path,
		},
	})
	c.Filesystem = append(c.Filesystem, service...)
}
