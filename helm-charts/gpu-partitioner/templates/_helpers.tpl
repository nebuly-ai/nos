{{/*
Expand the name of the chart.
*/}}
{{- define "gpu-partitioner.name" -}}
{{- printf "%s-%s" .Values.namePrefix .Chart.Name | trunc 63 }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "gpu-partitioner.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "gpu-partitioner.labels" -}}
helm.sh/chart: {{ include "gpu-partitioner.chart" . }}
{{ include "gpu-partitioner.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "gpu-partitioner.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gpu-partitioner.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: {{ "nebulnetes" }}
{{- end }}

{{/*
Create the name of the gpu partitioner config ConfigMap
*/}}
{{- define "gpu-partitioner.config.configMapName" -}}
{{- include "gpu-partitioner.name" . }}-config
{{- end }}

{{/*
Create the name of the known MIG geometries ConfigMap
*/}}
{{- define "gpu-partitioner.knownMigGeometriesConfigMapName" -}}
{{- include "gpu-partitioner.name" . }}-known-mig-geometries
{{- end }}

{{/*
Create the name of the file storing the possible MIG geometries of each known GPU model
*/}}
{{- define "gpu-partitioner.knownMigGeometriesFileName" -}}
known_mig_geometries.yaml
{{- end }}

{{/*
Create the name of the file storing the GPU Partitioner configuration
*/}}
{{- define "gpu-partitioner.configFileName" -}}
gpu_partitioner_config.yaml
{{- end }}



