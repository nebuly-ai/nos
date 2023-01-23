{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "nos.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Define nos full name including the Chart release name
*/}}
{{- define "nos.fullname" -}}
{{- $name := .Chart.Name -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- (printf "%s-%s" .Release.Name $name) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Nos labels
*/}}
{{- define "nos.labels" -}}
helm.sh/chart: {{ include "nos.chart" . }}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}


{{- define "nos.metricsConfigMap.name" -}}
{{- printf "%s-metrics" (include "nos.fullname" .) -}}
{{- end -}}

{{- define "nos.installationInfoConfigMap.name" -}}
{{- printf "%s-installation-info" (include "nos.fullname" .) -}}
{{- end -}}

{{/*{{- define "nos.installationUUID" -}}*/}}
{{/*{{- with lookup "v1" "ConfigMap" .Release.Namespace (include "nos.installationInfoConfigMap.name" . ) }}*/}}
{{/*{{- .data.uuid -}}*/}}
{{/*{{- else -}}*/}}
{{/*{{- uuidv4 -}}*/}}
{{/*{{- end -}}*/}}
{{/*{{- end}}*/}}


