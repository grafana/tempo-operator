package version

import (
	"fmt"
	"runtime"
)

var (
	version   string
	buildDate string
	tempo     string
)

// Version holds this Operator's version as well as the version of some of the components it uses.
type Version struct {
	Operator  string `json:"tempo-operator-version"`
	BuildDate string `json:"build-date"`
	Tempo     string `json:"tempo-version"`
	Go        string `json:"go-version"`
	GitHash	string `json:"commit-hash"`
}

// Get returns the Version object with the relevant information.
func Get() Version {
	return Version{
		Operator:  version,
		BuildDate: buildDate,
		Tempo:     Tempo(),
		Go:        runtime.Version(),
	}
}

func (v Version) String() string {
	return fmt.Sprintf(
		"Version(Operator='%v', BuildDate='%v', Tempo='%v', Go='%v')",
		v.Operator,
		v.BuildDate,
		v.Tempo,
		v.Go,
	)
}

// Tempo returns the default Tempo to use when no versions are specified via CLI or configuration.
func Tempo() string {
	if len(tempo) > 0 {
		// this should always be set, as it's specified during the build
		return tempo
	}

	// fallback value, useful for tests
	return "0.0.0"
}
