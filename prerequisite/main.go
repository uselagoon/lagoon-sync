package prerequisite

func init() {

	// run the prerequsite gatherers
	// prerequisiteConfig := prerequisite.GetConfigPrerequisite()
	// var rsyncPrerequisites []prerequisite.GatheredPrerequisite
	// var envVarPrerequisites []prerequisite.GatheredPrerequisite
	// var otherPrerequisites []prerequisite.GatheredPrerequisite

	// for _, c := range prerequisiteConfig {
	// 	if c.GetValue() {
	// 		gatheredConfig, err := c.GatherValue()
	// 		if err != nil {
	// 			log.Println(err.Error())
	// 			continue
	// 		}

	// 		switch c.GetName() {
	// 		case "rsync_path":
	// 			rsyncPrerequisites = append(rsyncPrerequisites, gatheredConfig...)
	// 		case "env-vars":
	// 			envVarPrerequisites = append(envVarPrerequisites, gatheredConfig...)
	// 		default:
	// 			otherPrerequisites = append(otherPrerequisites, gatheredConfig...)
	// 		}
	// 	}
	// }

}
