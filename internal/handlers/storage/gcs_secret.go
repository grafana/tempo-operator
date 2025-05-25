package storage

import (
	"encoding/json"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	"github.com/grafana/tempo-operator/internal/manifests/manifestutils"
)

const (
	bucketNameKey          = "bucketname"
	authFileKey            = "key.json"
	gcpAccountTypeExternal = "external_account"
)

func extractGoogleCredentialSource(secret *corev1.Secret) (sourceFile, sourceType string, err error) {
	keyJSON := secret.Data["key.json"]
	if len(keyJSON) == 0 {
		return "", "", errors.New("missing secret field key.json")
	}

	credentialsFile := struct {
		CredentialsType   string `json:"type"`
		CredentialsSource struct {
			File string `json:"file"`
		} `json:"credential_source"`
	}{}

	err = json.Unmarshal(keyJSON, &credentialsFile)
	if err != nil {
		return "", "", errors.New("gcp storage secret cannot be parsed from JSON content")
	}

	return credentialsFile.CredentialsSource.File, credentialsFile.CredentialsType, nil
}

func discoverGCSCredentialType(storageSecret corev1.Secret, path *field.Path) (v1alpha1.CredentialMode, field.ErrorList) {
	// Check if correct credential source is used
	_, credentialType, err := extractGoogleCredentialSource(&storageSecret)
	if err != nil {
		return "", field.ErrorList{field.Invalid(
			path,
			storageSecret.Name,
			err.Error(),
		)}
	}

	if credentialType == gcpAccountTypeExternal {
		return v1alpha1.CredentialModeToken, nil
	}

	return v1alpha1.CredentialModeStatic, nil

}

func validateGCSSecret(storageSecret corev1.Secret, path *field.Path, credentialMode v1alpha1.CredentialMode) field.ErrorList {
	switch credentialMode {
	case v1alpha1.CredentialModeStatic:
	case v1alpha1.CredentialModeToken:
		err := ensureNotEmpty(storageSecret, []string{bucketNameKey, authFileKey}, path)
		if err != nil {
			return err
		}
		credentialSource, _, errr := extractGoogleCredentialSource(&storageSecret)
		if errr != nil {
			return field.ErrorList{field.Invalid(
				path,
				storageSecret.Name,
				errr.Error(),
			)}
		}

		if credentialSource != manifestutils.ServiceAccountTokenFilePath {
			return field.ErrorList{field.Invalid(
				path,
				storageSecret.Name,
				"credential source in secret needs to point to token file",
			)}
		}
	case v1alpha1.CredentialModeTokenCCO:
		return field.ErrorList{field.Invalid(
			path,
			credentialMode,
			"credential mode not supported for GCS",
		)}
	}
	return field.ErrorList{}
}

func getGCSParams(storageSecret corev1.Secret, path *field.Path, mode v1alpha1.CredentialMode) (*manifestutils.GCS, field.ErrorList) {

	errs := validateGCSSecret(storageSecret, path, mode)
	if len(errs) != 0 {
		return nil, errs
	}

	if mode == v1alpha1.CredentialModeToken {
		audience := manifestutils.GcpDefaultAudience
		if aud, ok := storageSecret.Data["audience"]; ok {
			audience = string(aud)
		}

		return &manifestutils.GCS{
			Bucket:            string(storageSecret.Data["bucketname"]),
			IAMServiceAccount: string(storageSecret.Data["iam_sa"]),
			ProjectID:         string(storageSecret.Data["iam_sa_project_id"]),
			Audience:          audience,
		}, nil
	}

	return &manifestutils.GCS{
		Bucket: string(storageSecret.Data["bucketname"]),
	}, nil
}
