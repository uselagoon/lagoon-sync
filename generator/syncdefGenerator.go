package generator

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/uselagoon/lagoon-sync/synchers"
	"strings"
	"text/template"
)

func BuildConfigStanzaFromServices(services []LagoonServiceDefinition) (string, error) {

	mariadbServices := []synchers.MariadbSyncRoot{}
	filesystemServices := []synchers.FilesSyncRoot{}
	postgresServices := []synchers.PostgresSyncRoot{}
	serviceCount := 0
	// we go through the service definitions and try to generate text for them
	for _, v := range services {
		switch v.ServiceType {
		case "mariadb-single":
			fallthrough
		case "mariadb-dbaas":
			fallthrough
		case "mariadb":
			sr, err := GenerateMariadbSyncRootFromService(v)
			if err != nil {
				return "", err
			}
			serviceCount += 1
			mariadbServices = append(mariadbServices, sr)
			break
		case "cli-persistent": //cli and cli-persisten
			sr, err := GenerateFilesSyncRootFromPersistentService(v)
			if err != nil {
				return "", err
			}
			serviceCount += 1
			filesystemServices = append(filesystemServices, sr)
			fallthrough // in case there are multiple volumes also defined in this cli
		case "cli":
			// now we generate from multivolume, if they're there
			srs, err := GenerateFilesSyncRootsFromServiceDefinition(v)
			if err != nil {
				return "", err
			}
			serviceCount += len(srs)
			filesystemServices = append(filesystemServices, srs...)
			break
		case "postgres-single":
			fallthrough
		case "postgres-dbaas":
			fallthrough
		case "postgres":
			sr, err := GeneratePgqlSyncRootFromService(v)
			if err != nil {
				return "", err
			}
			serviceCount += 1
			postgresServices = append(postgresServices, sr)
		}
	}

	if serviceCount == 0 {
		return "", errors.New("No sync definitions were able to be extracted from service list")
	}

	allServices := struct {
		Mariadb    []synchers.MariadbSyncRoot
		Filesystem []synchers.FilesSyncRoot
		Postgres   []synchers.PostgresSyncRoot
	}{
		Mariadb:    mariadbServices,
		Filesystem: filesystemServices,
		Postgres:   postgresServices,
	}

	const yamlTemplate = `
lagoon-sync:
{{- range .Mariadb }}
  - {{ .ServiceName }}:
    type: {{ .Type }}
    config:
      hostname: "{{ .Config.DbHostname }}"
      username: "{{ .Config.DbUsername }}"
      password: "{{ .Config.DbPassword }}"
      port:     "{{ .Config.DbPort }}"
      database: "{{ .Config.DbDatabase }}"
{{- end }}
{{- range .Postgres }}
  - {{ .ServiceName }}:
    type: {{ .Type }}
    config:
      hostname: "{{ .Config.DbHostname }}"
      username: "{{ .Config.DbUsername }}"
      password: "{{ .Config.DbPassword }}"
      port:     "{{ .Config.DbPort }}"
      database: "{{ .Config.DbDatabase }}"
{{- end }}
{{- range .Filesystem }}
  - {{ .ServiceName }}:
    type: {{ .Type }}
      config:
        sync-directory: "{{ .Config.SyncPath }}"
{{- end }}
`
	// Parse and execute the template
	tmpl, err := template.New("yaml").Parse(yamlTemplate)
	if err != nil {
		panic(err)
	}

	var output bytes.Buffer
	err = tmpl.Execute(&output, allServices)
	if err != nil {
		panic(err)
	}

	return output.String(), nil
}

func GenerateMariadbSyncRootFromService(definition LagoonServiceDefinition) (synchers.MariadbSyncRoot, error) {

	// the main configuration detail we're interested in here is really the defaults for the host etc.
	serviceNameUppercase := strings.ToUpper(definition.ServiceName)

	syncRoot := synchers.MariadbSyncRoot{
		Type:        synchers.MariadbSyncPlugin{}.GetPluginId(),
		ServiceName: definition.ServiceName,
		Config:      synchers.BaseMariaDbSync{},
	}
	syncRoot.Config.SetDefaults()

	if serviceNameUppercase != strings.ToUpper(synchers.MariadbSyncPlugin{}.GetPluginId()) {
		syncRoot.Config.DbHostname = fmt.Sprintf("${%v_HOST:-mariadb}", serviceNameUppercase)
		syncRoot.Config.DbUsername = fmt.Sprintf("${%v_USERNAME:-drupal}", serviceNameUppercase)
		syncRoot.Config.DbPassword = fmt.Sprintf("${%v_PASSWORD:-drupal}", serviceNameUppercase)
		syncRoot.Config.DbPort = fmt.Sprintf("${%v_PORT:-3306}", serviceNameUppercase)
		syncRoot.Config.DbDatabase = fmt.Sprintf("${%v_DATABASE:-drupal}", serviceNameUppercase)
	}

	return syncRoot, nil
}

func GeneratePgqlSyncRootFromService(definition LagoonServiceDefinition) (synchers.PostgresSyncRoot, error) {

	// the main configuration detail we're interested in here is really the defaults for the host etc.
	serviceNameUppercase := strings.ToUpper(definition.ServiceName)

	syncRoot := synchers.PostgresSyncRoot{
		Type:        synchers.PostgresSyncPlugin{}.GetPluginId(),
		ServiceName: definition.ServiceName,
		Config:      synchers.BasePostgresSync{},
	}

	syncRoot.Config.SetDefaults()

	if serviceNameUppercase != strings.ToUpper(synchers.PostgresSyncPlugin{}.GetPluginId()) {
		syncRoot.Config.DbHostname = fmt.Sprintf("${%v_HOST:-postgres}", serviceNameUppercase)
		syncRoot.Config.DbUsername = fmt.Sprintf("${%v_USERNAME:-drupal}", serviceNameUppercase)
		syncRoot.Config.DbPassword = fmt.Sprintf("${%v_PASSWORD:-drupal}", serviceNameUppercase)
		syncRoot.Config.DbPort = fmt.Sprintf("${%v_PORT:-5432}", serviceNameUppercase)
		syncRoot.Config.DbDatabase = fmt.Sprintf("${%v_DATABASE:-drupal}", serviceNameUppercase)
	}

	return syncRoot, nil
}

func GenerateFilesSyncRootFromPersistentService(definition LagoonServiceDefinition) (synchers.FilesSyncRoot, error) {
	syncRoot := synchers.FilesSyncRoot{
		ServiceName: definition.ServiceName,
		Type:        synchers.FilesSyncPlugin{}.GetPluginId(),
		Config:      synchers.BaseFilesSync{},
	}
	v, exists := definition.Labels["lagoon.persistent"]

	if !exists {
		return syncRoot, errors.New("Could not find the 'lagoon.persistent' label in service: " + definition.ServiceName)
	}

	syncRoot.Config.SyncPath = v

	return syncRoot, nil
}

func GenerateFilesSyncRootsFromServiceDefinition(definition LagoonServiceDefinition) ([]synchers.FilesSyncRoot, error) {
	syncRoots := []synchers.FilesSyncRoot{}
	for k, v := range definition.Labels {
		labelParts := strings.Split(k, ".")
		if len(labelParts) == 4 && labelParts[0] == "lagoon" && labelParts[1] == "volumes" && labelParts[3] == "path" {
			syncRoot := synchers.FilesSyncRoot{
				ServiceName: fmt.Sprintf("%v-%v", definition.ServiceName, labelParts[2]),
				Type:        "files",
				Config: synchers.BaseFilesSync{
					SyncPath: v,
				},
			}

			syncRoots = append(syncRoots, syncRoot)

		}
	}
	return syncRoots, nil
}
