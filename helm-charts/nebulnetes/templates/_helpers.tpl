{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "n8s.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Define nebulnetes full name including the Chart release name
*/}}
{{- define "n8s.fullname" -}}
{{- $name := .Chart.Name -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- (printf "%s-%s" .Release.Name $name) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Create the name of the controller manager leader election role
*/}}
{{- define "n8s.leaderElectionRoleName" -}}
{{ include "n8s.fullname" . }}-leader-election
{{- end }}

{{/*
Create the name of the controller manager auth proxy role
*/}}
{{- define "n8s.authProxyRoleName" -}}
{{ include "n8s.fullname" . }}-auth-proxy
{{- end }}

{{/*
Create the name of the controller manager metrics reader role
*/}}
{{- define "n8s.metricsReaderRoleName" -}}
{{ include "n8s.fullname" . }}-metrics-reader
{{- end }}

{{/*
Create the name of the self-signed certificate issuer
*/}}
{{- define "n8s.selfSignedCertIssuerName" -}}
{{ include "n8s.fullname" . }}-self-signed-issuer
{{- end }}
