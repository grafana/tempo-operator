package manifestutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAWSPartitionForRegion(t *testing.T) {
	tests := []struct {
		name              string
		region            string
		expectedID        string
		expectedDNSSuffix string
		expectedS3        string
		expectedSTS       string
		expectCommercial  bool
	}{
		{
			name:              "commercial region",
			region:            "us-east-1",
			expectedID:        "aws",
			expectedDNSSuffix: "amazonaws.com",
			expectedS3:        "s3.us-east-1.amazonaws.com",
			expectedSTS:       "sts.us-east-1.amazonaws.com",
			expectCommercial:  true,
		},
		{
			name:              "govcloud region",
			region:            "us-gov-west-1",
			expectedID:        "aws-us-gov",
			expectedDNSSuffix: "amazonaws.com",
			expectedS3:        "s3.us-gov-west-1.amazonaws.com",
			expectedSTS:       "sts.us-gov-west-1.amazonaws.com",
			expectCommercial:  true,
		},
		{
			name:              "iso region",
			region:            "us-iso-east-1",
			expectedID:        "aws-iso",
			expectedDNSSuffix: "c2s.ic.gov",
			expectedS3:        "s3.us-iso-east-1.c2s.ic.gov",
			expectedSTS:       "sts.us-iso-east-1.c2s.ic.gov",
			expectCommercial:  false,
		},
		{
			// us-isob- must not be matched by the us-iso- prefix.
			name:              "isob region",
			region:            "us-isob-east-1",
			expectedID:        "aws-iso-b",
			expectedDNSSuffix: "sc2s.sgov.gov",
			expectedS3:        "s3.us-isob-east-1.sc2s.sgov.gov",
			expectedSTS:       "sts.us-isob-east-1.sc2s.sgov.gov",
			expectCommercial:  false,
		},
		{
			name:              "isof region",
			region:            "us-isof-south-1",
			expectedID:        "aws-iso-f",
			expectedDNSSuffix: "csp.hci.ic.gov",
			expectedS3:        "s3.us-isof-south-1.csp.hci.ic.gov",
			expectedSTS:       "sts.us-isof-south-1.csp.hci.ic.gov",
			expectCommercial:  false,
		},
		{
			name:              "isoe region",
			region:            "eu-isoe-west-1",
			expectedID:        "aws-iso-e",
			expectedDNSSuffix: "cloud.adc-e.uk",
			expectedS3:        "s3.eu-isoe-west-1.cloud.adc-e.uk",
			expectedSTS:       "sts.eu-isoe-west-1.cloud.adc-e.uk",
			expectCommercial:  false,
		},
		{
			name:              "european sovereign cloud region",
			region:            "eusc-de-east-1",
			expectedID:        "aws-eusc",
			expectedDNSSuffix: "amazonaws.eu",
			expectedS3:        "s3.eusc-de-east-1.amazonaws.eu",
			expectedSTS:       "sts.eusc-de-east-1.amazonaws.eu",
			expectCommercial:  false,
		},
		{
			name:              "china region",
			region:            "cn-north-1",
			expectedID:        "aws-cn",
			expectedDNSSuffix: "amazonaws.com.cn",
			expectedS3:        "s3.cn-north-1.amazonaws.com.cn",
			expectedSTS:       "sts.cn-north-1.amazonaws.com.cn",
			expectCommercial:  false,
		},
		{
			// A region added by AWS in the future must not break reconciliation.
			name:              "unknown region falls back to the commercial partition",
			region:            "mars-west-1",
			expectedID:        "aws",
			expectedDNSSuffix: "amazonaws.com",
			expectedS3:        "s3.mars-west-1.amazonaws.com",
			expectedSTS:       "sts.mars-west-1.amazonaws.com",
			expectCommercial:  true,
		},
		{
			name:              "empty region falls back to the commercial partition",
			region:            "",
			expectedID:        "aws",
			expectedDNSSuffix: "amazonaws.com",
			expectedS3:        "s3..amazonaws.com",
			expectedSTS:       "sts..amazonaws.com",
			expectCommercial:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partition := AWSPartitionForRegion(tt.region)

			assert.Equal(t, tt.expectedID, partition.ID)
			assert.Equal(t, tt.expectedDNSSuffix, partition.DNSSuffix)
			assert.Equal(t, tt.expectedS3, partition.S3Endpoint(tt.region))
			assert.Equal(t, tt.expectedSTS, partition.STSEndpoint(tt.region))
			assert.Equal(t, tt.expectCommercial, partition.IsCommercial())
		})
	}
}

func TestAWSPartitionForARN(t *testing.T) {
	tests := []struct {
		name       string
		arn        string
		expectedID string
		expectedOK bool
	}{
		{
			name:       "commercial role arn",
			arn:        "arn:aws:iam::123456789012:role/tempo",
			expectedID: "aws",
			expectedOK: true,
		},
		{
			name:       "govcloud role arn",
			arn:        "arn:aws-us-gov:iam::123456789012:role/tempo",
			expectedID: "aws-us-gov",
			expectedOK: true,
		},
		{
			name:       "iso role arn",
			arn:        "arn:aws-iso:iam::123456789012:role/tempo",
			expectedID: "aws-iso",
			expectedOK: true,
		},
		{
			name:       "isob role arn",
			arn:        "arn:aws-isob:iam::123456789012:role/tempo",
			expectedID: "aws",
			expectedOK: false,
		},
		{
			name:       "iso-b role arn",
			arn:        "arn:aws-iso-b:iam::123456789012:role/tempo",
			expectedID: "aws-iso-b",
			expectedOK: true,
		},
		{
			name:       "empty arn",
			arn:        "",
			expectedID: "aws",
			expectedOK: false,
		},
		{
			name:       "not an arn",
			arn:        "tempo-role",
			expectedID: "aws",
			expectedOK: false,
		},
		{
			name:       "truncated arn",
			arn:        "arn:aws-iso",
			expectedID: "aws",
			expectedOK: false,
		},
		{
			name:       "arn without partition",
			arn:        "arn::iam::123456789012:role/tempo",
			expectedID: "aws",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partition, ok := AWSPartitionForARN(tt.arn)

			assert.Equal(t, tt.expectedOK, ok)
			assert.Equal(t, tt.expectedID, partition.ID)
		})
	}
}
