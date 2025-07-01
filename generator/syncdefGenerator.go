package generator

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/uselagoon/lagoon-sync/synchers"
)

type configTemplateData struct {
	Mariadb    []synchers.MariadbSyncRoot
	Filesystem []synchers.FilesSyncRoot
	Postgres   []synchers.PostgresSyncRoot
	SshHost    string
	SshPort    string
	Api        string
}

func BuildConfigStanzaFromServices(services []LagoonServiceDefinition) (string, error) {

	mariadbServices := []synchers.MariadbSyncRoot{}
	filesystemServices := []synchers.FilesSyncRoot{}
	postgresServices := []synchers.PostgresSyncRoot{}
	serviceCount := 0
	// we go through the service definitions and try to generate text for them
	for _, v := range services {
		switch v.ServiceType {
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
		case "mariadb", "mariadb-single", "mariadb-dbaas":
			sr, err := GenerateMariadbSyncRootFromService(v)
			if err != nil {
				return "", err
			}
			serviceCount += 1
			mariadbServices = append(mariadbServices, sr)
			break
		case "postgres", "postgres-single", "postgres-dbaas":
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

	templateData := configTemplateData{
		Mariadb:    mariadbServices,
		Filesystem: filesystemServices,
		Postgres:   postgresServices,
	}

	retString, err := generateSyncStanza(templateData)
	if err != nil {
		log.Fatal(err)
	}

	return retString, nil
}

func generateSyncStanza(templateData configTemplateData) (string, error) {
	const yamlTemplate = `
# Below is your configuration for lagoon-sync.
# These data can live in either a separate .lagoon-sync.yml file
# or your .lagoon.yml file.

# If your project is on anything except the amazeeio cluster, which are the defaults
# and you're running lagoon-sync from a local container, you will have to configure
# the SSH and API endpoints. You can grab this information from running the lagoon
# cli's "lagoon config list" which will output the ssh endpoints and ports you need.
# Typically, though, this information is also available in the environment variables
# LAGOON_CONFIG_SSH_HOST and LAGOON_CONFIG_SSH_PORT.
# 
# These, for instance, are the amazeeio defaults
# api: https://api.lagoon.amazeeio.cloud/graphql
# ssh:
#   host: ssh.lagoon.amazeeio.cloud
#   port: "32222"

{{- if ne .Api ""}}
api: {{ .Api }}
{{- end}}

lagoon-sync:
{{- if ne .SshHost ""}}
  ssh:
    host: {{ .SshHost }}
    {{- if ne .SshPort ""}}
    port: "{{ .SshPort }}"
    {{- end}}
{{- end}}
{{- range .Mariadb }}
  {{ .ServiceName }}:
    type: {{ .Type }}
    config:
      hostname: "{{ .Config.DbHostname }}"
      username: "{{ .Config.DbUsername }}"
      password: "{{ .Config.DbPassword }}"
      port:     "{{ .Config.DbPort }}"
      database: "{{ .Config.DbDatabase }}"
{{- end }}
{{- range .Postgres }}
  {{ .ServiceName }}:
    type: {{ .Type }}
    config:
      hostname: "{{ .Config.DbHostname }}"
      username: "{{ .Config.DbUsername }}"
      password: "{{ .Config.DbPassword }}"
      port:     "{{ .Config.DbPort }}"
      database: "{{ .Config.DbDatabase }}"
{{- end }}
{{- range .Filesystem }}
  {{ .ServiceName }}:
    type: {{ .Type }}
    config:
      sync-directory: "{{ .Config.SyncPath }}"
{{- end }}
`
	// Parse and execute the template
	tmpl, err := template.New("yaml").Parse(yamlTemplate)
	if err != nil {
		return "", err
	}

	var output bytes.Buffer
	err = tmpl.Execute(&output, templateData)
	if err != nil {
		return "", err
	}

	retString := output.String()
	return retString, nil
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

	// now we try to infer the defaults for password and username
	defaultUser := ""
	defaultHost := ":-" + definition.ServiceName
	defaultPassword := ""
	defaultDatabase := ""
	if definition.image != "" && strings.Contains(definition.image, "uselagoon/mariadb") {

		if strings.Contains(definition.image, "drupal") {
			defaultUser = ":-drupal"
			defaultDatabase = ":-drupal"
			defaultPassword = ":-drupal"
		} else {
			defaultUser = ":-lagoon"
			defaultDatabase = ":-lagoon"
			defaultPassword = ":-lagoon"
		}

	}

	syncRoot.Config.DbHostname = fmt.Sprintf("${%v_HOST%v}", serviceNameUppercase, defaultHost)
	syncRoot.Config.DbUsername = fmt.Sprintf("${%v_USERNAME%v}", serviceNameUppercase, defaultUser)
	syncRoot.Config.DbPassword = fmt.Sprintf("${%v_PASSWORD%v}", serviceNameUppercase, defaultPassword)
	syncRoot.Config.DbPort = fmt.Sprintf("${%v_PORT:-3306}", serviceNameUppercase)
	syncRoot.Config.DbDatabase = fmt.Sprintf("${%v_DATABASE%v}", serviceNameUppercase, defaultDatabase)

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
