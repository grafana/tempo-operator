package cluster

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/cmd/gather/config"

	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	networkingv1 "k8s.io/api/networking/v1"
	policy1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const defaultDirectoryPermissions = 0750

type cluster struct {
	config               *config.Config
	apiAvailabilityCache map[schema.GroupVersionResource]bool
}

// NewCluster creates a new cluster.
func NewCluster(cfg *config.Config) cluster {
	return cluster{
		config:               cfg,
		apiAvailabilityCache: make(map[schema.GroupVersionResource]bool),
	}
}

func (c *cluster) getOperatorNamespace() (string, error) {
	if c.config.OperatorNamespace != "" {
		return c.config.OperatorNamespace, nil
	}

	deployment, err := c.getOperatorDeployment()
	if err != nil {
		return "", err
	}

	c.config.OperatorNamespace = deployment.Namespace

	return c.config.OperatorNamespace, nil
}

func (c *cluster) getOperatorDeployment() (appsv1.Deployment, error) {
	operatorDeployments := appsv1.DeploymentList{}
	err := c.config.KubernetesClient.List(context.TODO(), &operatorDeployments, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"app.kubernetes.io/name": "tempo-operator",
		}),
	})

	if err != nil {
		return appsv1.Deployment{}, err
	}

	if len(operatorDeployments.Items) == 0 {
		return appsv1.Deployment{}, fmt.Errorf("operator not found")
	}

	return operatorDeployments.Items[0], nil

}

// GetOperatorLogs gets the operator logs from the cluster.
func (c *cluster) GetOperatorLogs() error {
	deployment, err := c.getOperatorDeployment()
	if err != nil {
		return err
	}

	labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelectorPreValidated()
	operatorPods := corev1.PodList{}
	err = c.config.KubernetesClient.List(context.TODO(), &operatorPods, &client.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return err
	}

	pod := operatorPods.Items[0]
	c.getPodLogs(pod.Name, pod.Namespace, "manager")
	return nil
}

func (c *cluster) getPodLogs(podName, namespace, container string) {
	pods := c.config.KubernetesClientSet.CoreV1().Pods(namespace)
	writeLogToFile(c.config.CollectionDir, podName, container, pods)
}

// GetOperatorDeploymentInfo gets the operator deployment info from the cluster.
func (c *cluster) GetOperatorDeploymentInfo() error {
	err := os.MkdirAll(c.config.CollectionDir, defaultDirectoryPermissions)
	if err != nil {
		return err
	}

	deployment, err := c.getOperatorDeployment()
	if err != nil {
		return err
	}

	writeToFile(c.config.CollectionDir, &deployment)

	return nil
}

// GetOLMInfo gets the OLM info from the cluster.
func (c *cluster) GetOLMInfo() error {
	if !c.isAPIAvailable(schema.GroupVersionResource{
		Group:    operatorsv1.SchemeGroupVersion.Group,
		Version:  operatorsv1.SchemeGroupVersion.Version,
		Resource: "Operator",
	}) {
		log.Println("OLM info not available")
		return nil
	}

	outputDir := filepath.Join(c.config.CollectionDir, "olm")
	err := os.MkdirAll(outputDir, defaultDirectoryPermissions)
	if err != nil {
		return err
	}

	operatorNamespace, err := c.getOperatorNamespace()
	if err != nil {
		return err
	}

	// Operators
	operators := operatorsv1.OperatorList{}
	err = c.config.KubernetesClient.List(context.TODO(), &operators, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range operators.Items {
		o := o
		writeToFile(outputDir, &o)

	}

	// OperatorGroups
	operatorGroups := operatorsv1.OperatorGroupList{}
	err = c.config.KubernetesClient.List(context.TODO(), &operatorGroups, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range operatorGroups.Items {
		o := o
		if strings.Contains(o.Name, "tempo") {
			writeToFile(outputDir, &o)
		}
	}

	// Subscription
	subscriptions := operatorsv1alpha1.SubscriptionList{}
	err = c.config.KubernetesClient.List(context.TODO(), &subscriptions, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range subscriptions.Items {
		o := o
		writeToFile(outputDir, &o)
	}

	// InstallPlan
	ips := operatorsv1alpha1.InstallPlanList{}
	err = c.config.KubernetesClient.List(context.TODO(), &ips, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range ips.Items {
		o := o
		writeToFile(outputDir, &o)
	}

	// ClusterServiceVersion
	csvs := operatorsv1alpha1.ClusterServiceVersionList{}
	err = c.config.KubernetesClient.List(context.TODO(), &csvs, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range csvs.Items {
		o := o
		if strings.Contains(o.Name, "tempo") {
			writeToFile(outputDir, &o)
		}
	}

	return nil
}

// GetTempoStacks gets all the TempoStacks in the cluster and resources owned by them.
func (c *cluster) GetTempoStacks() error {
	tempoStacks := tempov1alpha1.TempoStackList{}

	err := c.config.KubernetesClient.List(context.TODO(), &tempoStacks)
	if err != nil {
		return err
	}

	log.Println("TempoStacks found:", len(tempoStacks.Items))

	errorDetected := false

	for _, tempoStack := range tempoStacks.Items {
		tempoStack := tempoStack
		err := c.processTempoStack(&tempoStack)
		if err != nil {
			log.Fatalln(err)
			errorDetected = true
		}
	}

	if errorDetected {
		return fmt.Errorf("something failed while getting the tempostacks")
	}
	return nil
}

// GetTempoMonolithics gets all the TempoMonolithics in the cluster and resources owned by them.
func (c *cluster) GetTempoMonolithics() error {
	tempoMonolithics := tempov1alpha1.TempoMonolithicList{}

	err := c.config.KubernetesClient.List(context.TODO(), &tempoMonolithics)
	if err != nil {
		return err
	}

	log.Println("TempoMonolithic found:", len(tempoMonolithics.Items))

	errorDetected := false

	for _, tempoMonolithic := range tempoMonolithics.Items {
		tempoMonolithic := tempoMonolithic
		err := c.processTempoMonolithic(&tempoMonolithic)
		if err != nil {
			log.Fatalln(err)
			errorDetected = true
		}
	}

	if errorDetected {
		return fmt.Errorf("something failed while getting the tempomonolithics")
	}
	return nil
}

func (c *cluster) processTempoStack(tempoStack *tempov1alpha1.TempoStack) error {
	log.Printf("Processing TempoStack %s/%s", tempoStack.Namespace, tempoStack.Name)
	folder, err := createTempoStackFolder(c.config.CollectionDir, tempoStack)
	if err != nil {
		return err
	}
	writeToFile(folder, tempoStack)

	err = c.processOwnedResources(tempoStack, folder)
	if err != nil {
		return err
	}

	return nil
}

func (c *cluster) processTempoMonolithic(tempoMonolithic *tempov1alpha1.TempoMonolithic) error {
	log.Printf("Processing TempoMonolithic %s/%s", tempoMonolithic.Namespace, tempoMonolithic.Name)
	folder, err := createTempoMonolithicFolder(c.config.CollectionDir, tempoMonolithic)
	if err != nil {
		return err
	}
	writeToFile(folder, tempoMonolithic)

	err = c.processOwnedResources(tempoMonolithic, folder)
	if err != nil {
		return err
	}

	return nil
}

func (c *cluster) processOwnedResources(owner interface{}, folder string) error {
	resourceTypes := []struct {
		list     client.ObjectList
		apiCheck func() bool
	}{
		{&appsv1.DaemonSetList{}, func() bool { return true }},
		{&appsv1.DeploymentList{}, func() bool { return true }},
		{&appsv1.StatefulSetList{}, func() bool { return true }},
		{&rbacv1.ClusterRoleList{}, func() bool { return true }},
		{&rbacv1.ClusterRoleBindingList{}, func() bool { return true }},
		{&corev1.ConfigMapList{}, func() bool { return true }},
		{&corev1.PersistentVolumeList{}, func() bool { return true }},
		{&corev1.PersistentVolumeClaimList{}, func() bool { return true }},
		{&corev1.PodList{}, func() bool { return true }},
		{&corev1.ServiceList{}, func() bool { return true }},
		{&corev1.ServiceAccountList{}, func() bool { return true }},
		{&autoscalingv2.HorizontalPodAutoscalerList{}, func() bool { return true }},
		{&networkingv1.IngressList{}, func() bool { return true }},
		{&policy1.PodDisruptionBudgetList{}, func() bool { return true }},
		{&monitoringv1.PodMonitorList{}, c.isMonitoringAPIAvailable},
		{&monitoringv1.ServiceMonitorList{}, c.isMonitoringAPIAvailable},
		{&routev1.RouteList{}, c.isRouteAPIAvailable},
	}

	for _, rt := range resourceTypes {
		if rt.apiCheck() {
			if err := c.processResourceType(rt.list, owner, folder); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *cluster) processResourceType(list client.ObjectList, owner interface{}, folder string) error {
	resources, err := c.getOwnerResources(list, owner)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}
	for _, resource := range resources {
		writeToFile(folder, resource)
	}
	return nil
}

func (c *cluster) isMonitoringAPIAvailable() bool {
	return c.isAPIAvailable(schema.GroupVersionResource{
		Group:    monitoringv1.SchemeGroupVersion.Group,
		Version:  monitoringv1.SchemeGroupVersion.Version,
		Resource: "ServiceMonitor",
	})
}

func (c *cluster) isRouteAPIAvailable() bool {
	return c.isAPIAvailable(schema.GroupVersionResource{
		Group:    routev1.GroupName,
		Version:  routev1.GroupVersion.Version,
		Resource: "Route",
	})
}

func (c *cluster) isAPIAvailable(gvr schema.GroupVersionResource) bool {
	if result, ok := c.apiAvailabilityCache[gvr]; ok {
		return result
	}

	rm := c.config.KubernetesClient.RESTMapper()

	gvk, err := rm.KindFor(gvr)
	result := err == nil && !gvk.Empty()
	c.apiAvailabilityCache[gvr] = result

	return result
}

func (c *cluster) getOwnerResources(objList client.ObjectList, owner interface{}) ([]client.Object, error) {
	err := c.config.KubernetesClient.List(context.TODO(), objList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"app.kubernetes.io/managed-by": "tempo-operator",
		}),
	})
	if err != nil {
		return nil, err
	}

	resources := []client.Object{}

	items := reflect.ValueOf(objList).Elem().FieldByName("Items")
	for i := 0; i < items.Len(); i++ {
		item := items.Index(i).Addr().Interface().(client.Object)
		if hasOwnerReference(item, owner) {
			resources = append(resources, item)
		}
	}
	return resources, nil

}

func hasOwnerReference(obj client.Object, owner interface{}) bool {
	var ownerKind string
	var ownerUID types.UID

	switch o := owner.(type) {
	case *tempov1alpha1.TempoStack:
		ownerKind = o.Kind
		ownerUID = o.UID
	case *tempov1alpha1.TempoMonolithic:
		ownerKind = o.Kind
		ownerUID = o.UID
	default:
		return false
	}

	for _, ownerRef := range obj.GetOwnerReferences() {
		if ownerRef.Kind == ownerKind && ownerRef.UID == ownerUID {
			return true
		}
	}
	return false
}
