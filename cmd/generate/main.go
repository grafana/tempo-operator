package generate

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configtempov1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/cmd"
	"github.com/os-observability/tempo-operator/internal/manifests"
)

func loadSpec(path string) (v1alpha1.Microservices, error) {
	file, err := os.Open(path)
	if err != nil {
		return v1alpha1.Microservices{}, err
	}
	defer file.Close()

	spec := v1alpha1.Microservices{}
	decoder := yaml.NewYAMLOrJSONDecoder(file, 8192)
	err = decoder.Decode(&spec)
	if err != nil {
		return v1alpha1.Microservices{}, err
	}

	return spec, nil
}

func build(ctrlConfig configtempov1alpha1.ProjectConfig, params manifests.Params) ([]client.Object, error) {
	// apply default values from Defaulter webhook
	defaulterWebhook := v1alpha1.NewDefaulter(ctrlConfig.DefaultImages)
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

func toYAML(scheme *runtime.Scheme, objects []client.Object, out io.Writer) error {
	encoder := json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{Yaml: true})

	for _, obj := range objects {
		fmt.Fprintln(out, "---")
		types, _, err := scheme.ObjectKinds(obj)
		if err != nil {
			return fmt.Errorf("error getting object kind: %v", err)
		}
		if len(types) == 0 {
			return fmt.Errorf("could not find object kind for %v", obj)
		}

		obj.GetObjectKind().SetGroupVersionKind(types[0])
		err = encoder.Encode(obj, out)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}
	}

	return nil
}

func generate(c *cobra.Command, crPath string, outPath string, params manifests.Params) error {
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

	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("error opening output file: %w", err)
	}
	defer outFile.Close()

	err = toYAML(options.Scheme, objects, outFile)
	if err != nil {
		return fmt.Errorf("error generating yaml: %w", err)
	}

	return nil
}

func NewGenerateCommand() *cobra.Command {
	var crPath string
	var outPath string
	params := manifests.Params{
		StorageParams: manifests.StorageParams{
			S3: manifests.S3{},
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
