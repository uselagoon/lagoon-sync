package prerequisite

import (
	"testing"
)

type AlwaysTrueGatherer struct {
}

func (p AlwaysTrueGatherer) handlesPrerequisite(name string) bool {
	return true
}

func (p AlwaysTrueGatherer) GatherPrerequisites() []GatheredPrerequisite {
	return []GatheredPrerequisite{}
}

func TestRegisterGatherer(t *testing.T) {
	type args struct {
		name     string
		gatherer PrerequisiteGatherer
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "always returns true test",
			args: args{
				name:     "alwaysreturnstrue",
				gatherer: AlwaysTrueGatherer{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterGatherer(tt.name, tt.args.gatherer)
			// prereqGathererMap = append(prereqGathererMap, tt.args.gatherer)

			if len(prereqGathererMap) > 0 {
				return
			}
			t.Errorf("%v", prereqGathererMap)
		})
	}
}
