package prerequisite

type ExamplePrerequisite struct {

}

func (receiver ExamplePrerequisite) GetName() string {
	return "example-prerequisite"
}

// HandlesPrerequsite will tell the prerequisite system if the current
// prerequisite is handled by this gatherer
func (receiver ExamplePrerequisite) HandlesPrerequisite(prerequisiteName string) bool {
	if(prerequisiteName == "example") {
		return true
	}
	return false
}

func (receiver ExamplePrerequisite) GatherPrerequisites() ([]GatheredPrerequisite, error) {
	return []GatheredPrerequisite{}, nil
}

func init()  {
	example := ExamplePrerequisite{}
	RegisterPrerequisiteGatherer(example.GetName(), example)
}