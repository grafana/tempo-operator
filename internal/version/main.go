package version

import (
	"fmt"
	"runtime"

	"github.com/os-observability/tempo-operator/apis/config/v1alpha1"
)

var (
	version   string
	buildDate string
	commitSha string
)

// Version holds this Operator's version as well as the version of some of the components it uses.
type Version struct {
	OperatorVersion        string `json:"tempo-operator-version"`
	BuildDate              string `json:"build-date"`
	DefaultTempoImage      string `json:"tempo-image"`
	DefaultTempoQueryImage string `json:"tempo-query-image"`
	Go                     string `json:"go-version"`
	GitHash                string `json:"commit-hash"`
}

// Get returns the Version object with the relevant information.
func Get(config v1alpha1.ProjectConfig) Version {
	v := Version{
		OperatorVersion:        version,
		BuildDate:              buildDate,
		DefaultTempoImage:      config.DefaultImages.Tempo,
		DefaultTempoQueryImage: config.DefaultImages.TempoQuery,
		Go:                     runtime.Version(),
		GitHash:                commitSha,
	}
	return v
}

func (v Version) String() string {
	return fmt.Sprintf(
		"Version(OperatorVersion='%v', BuildDate='%v', DefaultTempoImage='%v', DefaultTempoQueryImage='%v', Go='%v', CommitHash='%v')",
		v.OperatorVersion,
		v.BuildDate,
		v.DefaultTempoImage,
		v.DefaultTempoQueryImage,
		v.Go,
		v.GitHash,
	)
}
