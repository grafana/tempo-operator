package root

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	configv1alpha1 "github.com/grafana/tempo-operator/api/config/v1alpha1"
)

var errConfigFileLoading = errors.New("could not read file at path")

func loadConfigFile(scheme *runtime.Scheme, outConfig *configv1alpha1.ProjectConfig, configFile string) error {
	content, err := os.ReadFile(filepath.Clean(configFile))
	if err != nil {
		return fmt.Errorf("%w %s", errConfigFileLoading, configFile)
	}

	codecs := serializer.NewCodecFactory(scheme)

	if err = runtime.DecodeInto(codecs.UniversalDecoder(), content, outConfig); err != nil {
		return fmt.Errorf("could not decode file into runtime.Object: %w", err)
	}

	return nil
}
