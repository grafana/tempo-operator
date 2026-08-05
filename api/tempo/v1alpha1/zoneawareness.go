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

// EffectiveReplicationFactor returns the replication factor of the stack, preferring spec.replication.factor
// over the deprecated spec.replicationFactor.
func (spec *TempoStackSpec) EffectiveReplicationFactor() int {
	if spec.Replication != nil && spec.Replication.Factor > 0 {
		return spec.Replication.Factor
	}
	return spec.ReplicationFactor
}

// ZoneAwarenessEnabled returns true if the stack is configured to spread its components across zones.
func (spec *TempoStackSpec) ZoneAwarenessEnabled() bool {
	return spec.Replication != nil && len(spec.Replication.Zones) > 0
}
