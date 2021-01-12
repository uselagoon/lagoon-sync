package prerequisite

// func RunPrerequisiteCommand(environment Environment, syncer Syncer, syncerType string, dryRun bool, verboseSSH bool) (Environment, error) {
// 	// We don't run prerequisite checks on these syncers for now.
// 	if syncerType == "files" || syncerType == "drupalconfig" {
// 		environment.RsyncPath = "rsync"
// 		return environment, nil
// 	}

// 	log.Printf("Running prerequisite checks on %s environment", environment.EnvironmentName)

// 	var execString string
// 	var configRespSuccessful bool

// 	command, commandErr := syncer.GetPrerequisiteCommand(environment, "config").GetCommand()
// 	if commandErr != nil {
// 		return environment, commandErr
// 	}

// 	if environment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
// 		execString = command
// 	} else {
// 		execString = generateRemoteCommand(environment, command, verboseSSH)
// 	}

// 	log.Printf("Running the following prerequisite command:- %s", execString)

// 	err, configResponseJson, errstring := Shellout(execString)
// 	if err != nil {
// 		fmt.Println(errstring)
// 		// return "", err
// 	}

// 	data := &PreRequisiteResponse{}
// 	json.Unmarshal([]byte(configResponseJson), &data)

// 	if !data.IsPrerequisiteResponseEmpty() {
// 		log.Printf("Config response: %v", configResponseJson)
// 		configRespSuccessful = true
// 	} else {
// 		log.Printf("%v\n-----\nWarning: lagoon-sync is not available on %s\n-----", configResponseJson, environment.EnvironmentName)
// 		configRespSuccessful = false
// 	}

// 	// Check if environment has rsync
// 	if data.RysncPrerequisite != nil {
// 		for _, c := range data.RysncPrerequisite {
// 			if c.Value != "" {
// 				environment.RsyncAvailable = true
// 				environment.RsyncPath = c.Value
// 			}
// 		}
// 	}

// 	lagoonSyncVersion := "unknown"
// 	if data.Version != "" {
// 		lagoonSyncVersion = data.Version
// 	}

// 	// Check if prerequisite checks were successful.
// 	if !configRespSuccessful {
// 		log.Printf("Unable to determine rsync config on %v, will attempt to use 'rsync' path instead", environment.EnvironmentName)
// 		environment.RsyncPath = "rsync"
// 		return environment, nil
// 	}
// 	if environment.RsyncAvailable {
// 		log.Printf("Rsync found: %v", environment.RsyncPath)
// 	}

// 	if !dryRun && !environment.RsyncAvailable {
// 		// Add rsync to env
// 		rsyncPath, err := createRsync(environment, syncer, lagoonSyncVersion)
// 		if err != nil {
// 			fmt.Println(errstring)
// 			return environment, err
// 		}

// 		log.Printf("Rsync path: %s", rsyncPath)
// 		environment.RsyncPath = rsyncPath
// 		return environment, nil
// 	}

// 	return environment, nil
//}

// func (p *PreRequisiteResponse) IsPrerequisiteResponseEmpty() bool {
// 	return reflect.DeepEqual(&PreRequisiteResponse{}, p)
// }

// func PrerequisiteCleanUp(environment Environment, rsyncPath string, dryRun bool, verboseSSH bool) error {
// 	log.Printf("Beginning prerequisite resource cleanup on %s", environment.EnvironmentName)
// 	if rsyncPath == "" || rsyncPath == "rsync" || !strings.Contains(rsyncPath, "/tmp/") {
// 		return nil
// 	}
// 	execString := fmt.Sprintf("rm -r %s", rsyncPath)

// 	if environment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
// 		execString = generateRemoteCommand(environment, execString, verboseSSH)
// 	}

// 	log.Printf("Running the following: %s", execString)

// 	if !dryRun {
// 		err, _, errstring := Shellout(execString)

// 		if err != nil {
// 			fmt.Println(errstring)
// 			return err
// 		}
// 	}

// 	return nil
// }
