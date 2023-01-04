{{/*
Expand the name of the chart.
*/}}
{{- define "scheduler.name" -}}
{{- printf "%s-%s" .Values.namePrefix "scheduler" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "scheduler.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "scheduler.labels" -}}
helm.sh/chart: {{ include "scheduler.chart" . }}
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
app.kubernetes.io/name: n8s-scheduler
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: {{ "nebulnetes" }}
{{- end }}

{{/*
Create the name of the scheduler config ConfigMap
*/}}
{{- define "scheduler.config.configMapName" -}}
{{- include "scheduler.name" . }}-config
{{- end }}
