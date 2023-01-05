{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "n8s.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the name of the controller manager leader election role
*/}}
{{- define "n8s.leaderElectionRoleName" -}}
{{ .Values.namePrefix }}-leader-election
{{- end }}

{{/*
Create the name of the controller manager auth proxy role
*/}}
{{- define "n8s.authProxyRoleName" -}}
{{ .Values.namePrefix }}-auth-proxy
{{- end }}

{{/*
Create the name of the controller manager metrics reader role
*/}}
{{- define "n8s.metricsReaderRoleName" -}}
{{ .Values.namePrefix }}-metrics-reader
{{- end }}

{{/*
Create the name of the self-signed certificate issuer
*/}}
{{- define "n8s.selfSignedCertIssuerName" -}}
{{ .Values.namePrefix }}-selfsigned-issuer
{{- end }}
