package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configtempov1alpha1 "github.com/os-observability/tempo-operator/apis/config/v1alpha1"
	"github.com/os-observability/tempo-operator/apis/tempo/v1alpha1"
	"github.com/os-observability/tempo-operator/internal/manifests"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

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

func toYAML(objects []client.Object, out io.Writer) error {
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

func usage() {
	println("usage: go run ./cmd/generate/main.go -cr path/to/microservices/cr.yaml")
}

func main() {
	var configFile string
	var crPath string
	params := manifests.Params{
		StorageParams: manifests.StorageParams{
			S3: manifests.S3{},
		},
	}

	flag.StringVar(&configFile, "config", "config/manager/controller_manager_config.yaml", "The controller configuration")
	flag.StringVar(&crPath, "cr", "", "Input CR")
	flag.StringVar(&params.StorageParams.S3.Endpoint, "storage.endpoint", "http://minio.minio.svc:9000", "S3 storage endpoint (taken from storage secret)")
	flag.StringVar(&params.StorageParams.S3.Bucket, "storage.bucket", "tempo", "S3 storage bucket (taken from storage secret)")
	flag.Parse()

	if crPath == "" {
		usage()
		os.Exit(1)
	}

	ctrlConfig := configtempov1alpha1.ProjectConfig{}
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		_, err := options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
		if err != nil {
			log.Fatalf("unable to load the config file: %v", err)
		}
	}

	spec, err := loadSpec(crPath)
	if err != nil {
		log.Fatalf("error loading spec: %v", err)
	}

	params.Tempo = spec
	objects, err := build(ctrlConfig, params)
	if err != nil {
		log.Fatalf("error building manifests: %v", err)
	}

	err = toYAML(objects, os.Stdout)
	if err != nil {
		log.Fatalf("error generating yaml: %v", err)
	}
}
