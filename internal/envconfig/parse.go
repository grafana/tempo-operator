package envconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var setupLog = ctrl.Log.WithName("setup")

// parsePositiveDuration parses a string into a metav1.Duration.
// Returns an error if the duration is invalid, negative, or zero.
// Use this for fields where zero is not a valid value (e.g., cert validity, lease duration).
func parsePositiveDuration(val string) (metav1.Duration, error) {
	d, err := time.ParseDuration(val)
	if err != nil {
		return metav1.Duration{}, err
	}
	if d <= 0 {
		return metav1.Duration{}, errors.New("duration must be positive")
	}
	return metav1.Duration{Duration: d}, nil
}

// lookupDurationEnv looks up an environment variable and parses it as a positive duration.
// Returns the duration and true if successful, or zero duration and false otherwise.
// Logs an error if the env var is set but has an invalid value.
func lookupDurationEnv(envName string) (metav1.Duration, bool) {
	val, ok := os.LookupEnv(envName)
	if !ok {
		return metav1.Duration{}, false
	}
	d, err := parsePositiveDuration(val)
	if err != nil {
		setupLog.Error(err, "invalid value for environment variable, ignoring", "env", envName, "value", val)
		return metav1.Duration{}, false
	}
	return d, true
}

// lookupBoolEnv looks up an environment variable and parses it as a boolean.
// Returns the value and true if successful, or false and false otherwise.
// Logs an error if the env var is set but has an invalid value.
func lookupBoolEnv(envName string) (bool, bool) {
	val, ok := os.LookupEnv(envName)
	if !ok {
		return false, false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		setupLog.Error(err, "invalid value for environment variable, ignoring", "env", envName, "value", val)
		return false, false
	}
	return b, true
}

// parsePodSecurityContext parses a JSON string into a PodSecurityContext.
// Returns an error if the JSON is invalid or contains unknown fields.
func parsePodSecurityContext(val string) (*corev1.PodSecurityContext, error) {
	psc := &corev1.PodSecurityContext{}
	decoder := json.NewDecoder(bytes.NewReader([]byte(val)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(psc); err != nil {
		return nil, err
	}
	return psc, nil
}
