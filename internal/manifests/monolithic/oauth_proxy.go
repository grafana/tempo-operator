package monolithic

import "github.com/grafana/tempo-operator/apis/tempo/v1alpha1"

func isOauthProxyEnabledForJaegerUI(tempo v1alpha1.TempoMonolithic) bool {
	return tempo.Spec.JaegerUI != nil && tempo.Spec.JaegerUI.Enabled && tempo.Spec.JaegerUI.Authentication != nil && tempo.Spec.JaegerUI.Authentication.Enabled
}

func isOauthProxyEnabledForTempo(tempo v1alpha1.TempoMonolithic) bool {
	return tempo.Spec.Query != nil && tempo.Spec.Query.Authentication != nil && tempo.Spec.Query.Authentication.Enabled
}

func isOauthProxyEnabled(tempo v1alpha1.TempoMonolithic) bool {
	return isOauthProxyEnabledForTempo(tempo) || isOauthProxyEnabledForJaegerUI(tempo)
}
