package version

import (
	"fmt"
	"runtime"

	promversion "github.com/prometheus/common/version"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	buildDate string
	revision  string

	operatorVersion   = "0.0.0"
	tempoVersion      = "0.0.0"
	tempoQueryVersion = "0.0.0"
)

// Version holds this Operator's version as well as the version of some of the components it uses.
type Version struct {
	BuildDate string `json:"build-date"`
	Revision  string `json:"revision"`

	OperatorVersion   string `json:"operator-version"`
	TempoVersion      string `json:"tempo-version"`
	TempoQueryVersion string `json:"tempo-query-version"`
	GoVersion         string `json:"go-version"`
}

func init() {
	promversion.Version = operatorVersion
	promversion.BuildDate = buildDate
	promversion.Revision = revision
	metrics.Registry.MustRegister(promversion.NewCollector("tempooperator"))
}

// Get returns the Version object with the relevant information.
func Get() Version {
	v := Version{
		BuildDate: buildDate,
		Revision:  revision,

		OperatorVersion:   operatorVersion,
		TempoVersion:      tempoVersion,
		TempoQueryVersion: tempoQueryVersion,
		GoVersion:         runtime.Version(),
	}
	return v
}

func (v Version) String() string {
	return fmt.Sprintf(
		"Version(BuildDate='%v', Revision='%v', OperatorVersion='%v', TempoVersion='%v', TempoQueryVersion='%v', GoVersion='%v')",
		v.BuildDate,
		v.Revision,
		v.OperatorVersion,
		v.TempoVersion,
		v.TempoQueryVersion,
		v.GoVersion,
	)
}
