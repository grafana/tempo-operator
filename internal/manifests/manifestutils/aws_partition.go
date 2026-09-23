package manifestutils

import (
	"fmt"
	"strings"
)

// AWSCommercialDNSSuffix is the DNS suffix of the service endpoints of the commercial
// and GovCloud AWS partitions.
const AWSCommercialDNSSuffix = "amazonaws.com"

// AWSPartition describes an AWS partition (commercial, GovCloud, ISO, ISOB, ...).
// AWS partitions are isolated from each other and expose their services under
// different DNS suffixes and ARN prefixes.
type AWSPartition struct {
	// ID is the partition identifier used in ARNs, for example aws-iso.
	ID string
	// DNSSuffix is the DNS suffix of the service endpoints of the partition,
	// for example c2s.ic.gov.
	DNSSuffix string
}

var awsCommercialPartition = AWSPartition{ID: "aws", DNSSuffix: AWSCommercialDNSSuffix}

// awsPartitions maps a region prefix to the partition the region belongs to.
// The order matters: the first matching prefix wins, therefore a prefix must be listed
// before any other prefix it extends (us-isob- before us-iso-).
var awsPartitions = []struct {
	regionPrefix string
	partition    AWSPartition
}{
	{"us-gov-", AWSPartition{ID: "aws-us-gov", DNSSuffix: AWSCommercialDNSSuffix}},
	{"us-isob-", AWSPartition{ID: "aws-iso-b", DNSSuffix: "sc2s.sgov.gov"}},
	{"us-isof-", AWSPartition{ID: "aws-iso-f", DNSSuffix: "csp.hci.ic.gov"}},
	{"us-iso-", AWSPartition{ID: "aws-iso", DNSSuffix: "c2s.ic.gov"}},
	{"eu-isoe-", AWSPartition{ID: "aws-iso-e", DNSSuffix: "cloud.adc-e.uk"}},
	{"eusc-", AWSPartition{ID: "aws-eusc", DNSSuffix: "amazonaws.eu"}},
	{"cn-", AWSPartition{ID: "aws-cn", DNSSuffix: "amazonaws.com.cn"}},
}

// AWSPartitionForRegion returns the AWS partition a region belongs to.
// Unknown and empty regions default to the commercial partition, so that a region
// added by AWS in the future does not break the reconciliation of existing stacks.
func AWSPartitionForRegion(region string) AWSPartition {
	for _, p := range awsPartitions {
		if strings.HasPrefix(region, p.regionPrefix) {
			return p.partition
		}
	}
	return awsCommercialPartition
}

// AWSPartitionForARN returns the partition encoded in the second field of an ARN
// (arn:<partition>:<service>:...). It reports false if the ARN cannot be parsed or
// names a partition this operator does not know.
func AWSPartitionForARN(arn string) (AWSPartition, bool) {
	parts := strings.SplitN(arn, ":", 3)
	if len(parts) < 3 || parts[0] != "arn" || parts[1] == "" {
		return awsCommercialPartition, false
	}

	if parts[1] == awsCommercialPartition.ID {
		return awsCommercialPartition, true
	}
	for _, p := range awsPartitions {
		if p.partition.ID == parts[1] {
			return p.partition, true
		}
	}
	return awsCommercialPartition, false
}

// S3Endpoint returns the regional S3 endpoint of the partition, without a scheme.
func (p AWSPartition) S3Endpoint(region string) string {
	return fmt.Sprintf("s3.%s.%s", region, p.DNSSuffix)
}

// STSEndpoint returns the regional STS endpoint of the partition, without a scheme.
func (p AWSPartition) STSEndpoint(region string) string {
	return fmt.Sprintf("sts.%s.%s", region, p.DNSSuffix)
}

// IsCommercial reports whether the partition serves its endpoints under the
// amazonaws.com DNS suffix, which is the case for the commercial and GovCloud partitions.
func (p AWSPartition) IsCommercial() bool {
	return p.DNSSuffix == AWSCommercialDNSSuffix
}
