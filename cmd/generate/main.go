package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/cmd"
	"github.com/os-observability/tempo-operator/internal/manifests"
	"github.com/os-observability/tempo-operator/internal/manifests/manifestutils"
)

// yamlOrJsonDecoderBufferSize determines how far into the stream
// the decoder will look to figure out whether this is a JSON stream.
const yamlOrJsonDecoderBufferSize = 8192

func loadSpec(path string) (v1alpha1.Microservices, error) {
	pathCleaned := filepath.Clean(path)
	file, err := os.Open(pathCleaned)
	if err != nil {
		return v1alpha1.Microservices{}, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalf("Error closing file %s: %v", pathCleaned, err)
		}
	}()

	spec := v1alpha1.Microservices{}
	decoder := k8syaml.NewYAMLOrJSONDecoder(file, yamlOrJsonDecoderBufferSize)
	err = decoder.Decode(&spec)
	if err != nil {
		return v1alpha1.Microservices{}, err
	}

	return spec, nil
}

func build(ctrlConfig configv1alpha1.ProjectConfig, params manifestutils.Params) ([]client.Object, error) {
	// apply default values from Defaulter webhook
	defaulterWebhook := v1alpha1.NewDefaulter(ctrlConfig)
	err := defaulterWebhook.Default(context.Background(), &params.Tempo)
	if err != nil {
		return nil, err
	}

	objects, err := manifests.BuildAll(params)
	if err != nil {
		return nil, err
	}

	return objects, nil
}

func toYAMLManifest(scheme *runtime.Scheme, objects []client.Object, out io.Writer) error {
	for _, obj := range objects {
		fmt.Fprintln(out, "---")

		// set Group, Version and Kind
		types, _, err := scheme.ObjectKinds(obj)
		if err != nil {
			return fmt.Errorf("error getting object kind: %v", err)
		}
		if len(types) == 0 {
			return fmt.Errorf("could not find object kind for %v", obj)
		}
		obj.GetObjectKind().SetGroupVersionKind(types[0])

		// Marshal to JSON first, to respect json tags in structs
		jsonBytes, err := json.Marshal(obj)
		if err != nil {
			return err
		}

		// Unmarshal into a map and remove status field
		// Use yaml.Unmarshal because it detects the correct number type,
		// whereas json.Unmarshal uses float64 for every number
		var jsonObj map[string]interface{}
		err = yaml.Unmarshal(jsonBytes, &jsonObj)
		if err != nil {
			return err
		}
		delete(jsonObj, "status")

		// Finally, marshal into yaml
		yamlBytes, err := yaml.Marshal(jsonObj)
		if err != nil {
			return err
		}

		_, err = out.Write(yamlBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func generate(c *cobra.Command, crPath string, outPath string, params manifestutils.Params) error {
	rootCmdConfig := c.Context().Value(cmd.RootConfigKey{}).(cmd.RootConfig)
	ctrlConfig, options := rootCmdConfig.CtrlConfig, rootCmdConfig.Options

	spec, err := loadSpec(crPath)
	if err != nil {
		return fmt.Errorf("error loading spec: %w", err)
	}

	params.Tempo = spec
	objects, err := build(ctrlConfig, params)
	if err != nil {
		return fmt.Errorf("error building manifests: %w", err)
	}

	outPathCleaned := filepath.Clean(outPath)
	outFile, err := os.OpenFile(outPathCleaned, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("error opening output file: %w", err)
	}
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Fatalf("Error closing file %s: %v", outPathCleaned, err)
		}
	}()

	err = toYAMLManifest(options.Scheme, objects, outFile)
	if err != nil {
		return fmt.Errorf("error generating yaml: %w", err)
	}

	return nil
}

// NewGenerateCommand returns a new generate command.
func NewGenerateCommand() *cobra.Command {
	var crPath string
	var outPath string
	params := manifestutils.Params{
		StorageParams: manifestutils.StorageParams{
			S3: manifestutils.S3{},
		},
	}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate YAML manifests from a Tempo CR",
		RunE: func(cmd *cobra.Command, args []string) error {
			return generate(cmd, crPath, outPath, params)
		},
	}
	cmd.Flags().StringVar(&crPath, "cr", "/dev/stdin", "Input CR")
	_ = cmd.MarkFlagRequired("cr")
	cmd.Flags().StringVar(&outPath, "output", "/dev/stdout", "File to store the manifests")
	cmd.Flags().StringVar(&params.StorageParams.S3.Endpoint, "storage.endpoint", "http://minio.minio.svc:9000", "S3 storage endpoint (taken from storage secret)")
	cmd.Flags().StringVar(&params.StorageParams.S3.Bucket, "storage.bucket", "tempo", "S3 storage bucket (taken from storage secret)")

	return cmd
}
