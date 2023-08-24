package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1alpha1 "github.com/grafana/tempo-operator/apis/config/v1alpha1"
	"github.com/grafana/tempo-operator/apis/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/cmd"
	controllers "github.com/grafana/tempo-operator/controllers/tempo"
	"github.com/grafana/tempo-operator/internal/manifests"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

// yamlOrJsonDecoderBufferSize determines how far into the stream
// the decoder will look to figure out whether this is a JSON stream.
const yamlOrJsonDecoderBufferSize = 8192

var log = ctrl.Log.WithName("generate")

func loadSpec(r io.Reader) (v1alpha1.TempoStack, error) {
	spec := v1alpha1.TempoStack{}
	decoder := k8syaml.NewYAMLOrJSONDecoder(r, yamlOrJsonDecoderBufferSize)
	err := decoder.Decode(&spec)
	if err != nil {
		return v1alpha1.TempoStack{}, err
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
		delete(jsonObj["metadata"].(map[interface{}]interface{}), "creationTimestamp")
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

	var specReader io.Reader
	if crPath == "/dev/stdin" {
		log.Info("reading from stdin")
		specReader = c.InOrStdin()
	} else {
		pathCleaned := filepath.Clean(crPath)
		file, err := os.Open(pathCleaned)
		if err != nil {
			return fmt.Errorf("error reading cr: %w", err)
		}

		specReader = file
		defer func() {
			if err := file.Close(); err != nil {
				log.Error(err, "error closing file", "path", pathCleaned)
			}
		}()
	}

	spec, err := loadSpec(specReader)
	if err != nil {
		return fmt.Errorf("error loading spec: %w", err)
	}

	params.Tempo = spec
	objects, err := build(ctrlConfig, params)
	if err != nil {
		return fmt.Errorf("error building manifests: %w", err)
	}

	var output io.Writer
	if outPath == "/dev/stdout" {
		output = c.OutOrStdout()
	} else {
		outPathCleaned := filepath.Clean(outPath)
		outFile, err := os.OpenFile(outPathCleaned, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("error opening output file: %w", err)
		}
		output = outFile
		defer func() {
			if err := outFile.Close(); err != nil {
				log.Error(err, "error closing file", "path", outPathCleaned)
			}
		}()
	}

	err = toYAMLManifest(options.Scheme, objects, output)
	if err != nil {
		return fmt.Errorf("error generating yaml: %w", err)
	}

	return nil
}

// NewGenerateCommand returns a new generate command.
func NewGenerateCommand() *cobra.Command {
	var crPath string
	var outPath string
	var azureContainer string
	var gcsBucket string
	var s3Endpoint string
	var s3Bucket string
	params := manifestutils.Params{}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate YAML manifests from a Tempo CR",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case azureContainer != "":
				params.StorageParams.AzureStorage = controllers.GetAzureParams(v1alpha1.TempoStack{}, &corev1.Secret{Data: map[string][]byte{
					"container": []byte(azureContainer),
				}})
			case gcsBucket != "":
				params.StorageParams.GCS = controllers.GetGCSParams(v1alpha1.TempoStack{}, &corev1.Secret{Data: map[string][]byte{
					"bucketname": []byte(gcsBucket),
				}})
			case s3Endpoint != "":
				params.StorageParams.S3 = controllers.GetS3Params(v1alpha1.TempoStack{}, &corev1.Secret{Data: map[string][]byte{
					"endpoint": []byte(s3Endpoint),
					"bucket":   []byte(s3Bucket),
				}})
			}
			return generate(cmd, crPath, outPath, params)
		},
	}
	cmd.Flags().StringVar(&crPath, "cr", "/dev/stdin", "Input CR")
	cmd.Flags().StringVar(&outPath, "output", "/dev/stdout", "File to store the manifests")
	cmd.Flags().StringVar(&azureContainer, "storage.azure.container", "azure", "Azure container(taken from storage secret)")
	cmd.Flags().StringVar(&gcsBucket, "storage.gcs.bucket", "tempo", "GCS storage bucket (taken from storage secret)")
	cmd.Flags().StringVar(&s3Endpoint, "storage.s3.endpoint", "http://minio.minio.svc:9000", "S3 storage endpoint (taken from storage secret)")
	cmd.Flags().StringVar(&s3Bucket, "storage.s3.bucket", "tempo", "S3 storage bucket (taken from storage secret)")
	return cmd
}
