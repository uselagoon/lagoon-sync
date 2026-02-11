package cmd

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-sync/utils"
)

var dockerComposeFile string
var sersyncListOnly bool

// We use this to filter the standard service types we can sync.
var supportedSynchableServicetypes = []string{
	"mariadb",
	"mariadb-single",
	"mariadb-dbaas",
	"postgres",
	"postgres-single",
	"postgres-dbaas",
}

var serviceCmd = &cobra.Command{
	Use:   "service-sync",
	Short: "Automated service based sync tool",
	Long:  `List or sync all services and their volumes from a docker-compose.yml file or services api`,
	Run:   servicesCommandRun,
}

func servicesCommandRun(cmd *cobra.Command, args []string) {
	// Default to docker-compose.yml in current directory if not specified
	path := dockerComposeFile
	if path == "" {
		path = "docker-compose.yml"
	}

	services, err := utils.LoadDockerCompose(path)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to load docker-compose file: %v", err), nil)
	}

	if sersyncListOnly {
		prettyPrintServiceOutput(services)
		return
	}

	runService, err := selectServiceFromList(services, "Select service to use to do the transfer (typically your 'cli' service)", []string{})
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	// ask whether the user wants to sync files or databases
	syncType := "databases"
	if len(runService.Volumes) > 0 {
		syncType, err = selectSyncType()
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
	}

	// Now we offer the final menu of services
	switch syncType {
	case ("files"):
		// let's select the volume to move
		selectedVolume, err := selectVolume(runService.Volumes)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
		fmt.Printf("Gonna sync %v \n", selectedVolume)

		// okay, now we can actually invoke the synch

	default:
		// let's select a DB service to transfer
		syncService, err := selectServiceFromList(services, "Select service to sync", supportedSynchableServicetypes)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
		fmt.Printf("Gonna sync %v\n", syncService.Name)
	}

}

func selectServiceFromList(services map[string]utils.Service, title string, filterList []string) (utils.Service, error) {

	options := []huh.Option[string]{}

	// build an ordered list of service names so selection is deterministic
	names := make([]string, 0, len(services))

	for name, _ := range services {
		names = append(names, name)
	}
	sort.Strings(names)

	// Collect `cli` options first (preserving alphabetical order), then others.
	cliOpts := make([]huh.Option[string], 0, 1)
	otherOpts := make([]huh.Option[string], 0, len(names))
	for _, name := range names {
		svc := services[name]
		opt := huh.NewOption(fmt.Sprintf("%v - %v", name, svc.Type), name)

		if len(filterList) > 0 {
			if !utils.SliceContains(filterList, svc.Type) {
				continue
			}
		}

		if svc.Type == "cli" {
			cliOpts = append(cliOpts, opt)
		} else {
			// right now the only other types we support are mariadb and postgres
			otherOpts = append(otherOpts, opt)
		}
	}

	options = append(cliOpts, otherOpts...)

	var selected string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title(title).Options(options...).Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return utils.Service{}, err
	}

	return services[selected], nil

}

func selectSyncType() (string, error) {
	options := []huh.Option[string]{
		huh.NewOption("Files", "files"),
		huh.NewOption("Databases", "databases"),
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select what to sync").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}
	return selected, nil
}

func selectVolume(volumeMap map[string]string) (string, error) {
	// volumes := make([]string, len(volumeMap))
	volumes := []string{}
	for _, v := range volumeMap {
		volumes = append(volumes, v)
	}
	sort.Strings(volumes)
	options := []huh.Option[string]{}
	for _, vol := range volumes {
		options = append(options, huh.NewOption(vol, vol))
	}

	var selected string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select which volume to sync").
				Options(options...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return "", err
	}
	return selected, nil
}

func prettyPrintServiceOutput(services map[string]utils.Service) {
	// Output the services
	for name, svc := range services {
		fmt.Printf("Service: %s\n", name)
		fmt.Printf("  Type: %s\n", svc.Type)
		if len(svc.Volumes) > 0 {
			fmt.Printf("  Volumes:\n")
			for vol, path := range svc.Volumes {
				fmt.Printf("    %s: %s\n", vol, path)
			}
		}
	}
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.Flags().StringVarP(&dockerComposeFile, "docker-compose-file", "f", "", "Path to docker-compose.yml (defaults to docker-compose.yml in current directory)")
	serviceCmd.Flags().BoolVarP(&sersyncListOnly, "list-only", "l", false, "only display service sync options (default false)")
}
