package v1alpha1

const (
	// AnnotationAvailabilityZone contains the availability zone used in the Tempo configuration of that pod.
	// It is automatically added to managed Pods by the operator, if needed.
	AnnotationAvailabilityZone = "tempo.grafana.com/availability-zone"

	// AnnotationAvailabilityZoneLabels contains a list of node-labels that are used to construct the availability
	// zone of the annotated Pod. It is used by the zone-awareness controller and automatically added to managed
	// Pods by the operator, if needed.
	AnnotationAvailabilityZoneLabels = "tempo.grafana.com/availability-zone-labels"

	// LabelZoneAwarePod is a pod-label that is added to Pods that should be reconciled by the zone-awareness
	// controller. It is automatically added to managed Pods by the operator, if needed.
	LabelZoneAwarePod = "tempo.grafana.com/zone-aware"
)

// ZoneAwarenessEnabled returns true if the stack is configured to spread its components across zones.
func (spec *TempoStackSpec) ZoneAwarenessEnabled() bool {
	return len(spec.ReplicationZones) > 0
}
