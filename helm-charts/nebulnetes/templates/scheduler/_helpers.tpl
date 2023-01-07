{{/*
Expand the name of the chart.
*/}}
{{- define "scheduler.name" -}}
{{- "scheduler" -}}
{{- end }}

{{- define "scheduler.fullname" -}}
{{- $name := include "scheduler.name" . -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- printf "%s-%s" .Chart.Name (.Release.Name | replace .Chart.Name $name | trunc 63 | trimSuffix "-") -}}
{{- else -}}
{{- (printf "%s-%s" .Release.Name $name) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "scheduler.labels" -}}
helm.sh/chart: {{ include "n8s.chart" . }}
{{ include "scheduler.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "scheduler.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: scheduler
{{- end }}

{{/*
Create the name of the scheduler config ConfigMap
*/}}
{{- define "scheduler.config.configMapName" -}}
{{- include "scheduler.fullname" . }}-config
{{- end }}
