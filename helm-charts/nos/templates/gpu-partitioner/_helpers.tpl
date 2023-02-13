{{/*
*********************************************************************
* GPU Partitioner
*********************************************************************
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "gpuPartitioner.name" -}}
{{- default (printf "%s-%s" .Chart.Name "gpu-partitioner") .Values.gpuPartitioner.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "gpuPartitioner.fullname" -}}
{{- if .Values.gpuPartitioner.fullnameOverride }}
{{- .Values.gpuPartitioner.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name "gpu-partitioner" | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "gpuPartitioner.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
GPU Partitioner labels
*/}}
{{- define "gpuPartitioner.labels" -}}
helm.sh/chart: {{ include "gpuPartitioner.chart" . }}
{{ include "gpuPartitioner.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ "nos" }}
app.kubernetes.io/component: gpu-partitioner
{{- end }}

{{/*
GPU Partitioner selector labels
*/}}
{{- define "gpuPartitioner.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gpuPartitioner.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the gpu partitioner config ConfigMap
*/}}
{{- define "gpuPartitioner.config.configMapName" -}}
{{- include "gpuPartitioner.fullname" . }}-config
{{- end }}

{{/*
Create the name of the known MIG geometries ConfigMap
*/}}
{{- define "gpuPartitioner.knownMigGeometriesConfigMapName" -}}
{{- include "gpuPartitioner.fullname" . }}-known-mig-geometries
{{- end }}

{{/*
Create the name of the file storing the possible MIG geometries of each known GPU model
*/}}
{{- define "gpuPartitioner.knownMigGeometriesFileName" -}}
known_mig_geometries.yaml
{{- end }}

{{/*
Create the name of the file storing the scheduler config
*/}}
{{- define "gpuPartitioner.schedulerConfigFileName" -}}
scheduler_config.yaml
{{- end }}

{{/*
Create the name of the file storing the GPU Partitioner configuration
*/}}
{{- define "gpuPartitioner.configFileName" -}}
gpu_partitioner_config.yaml
{{- end }}

{{/*
Create the name of the controller manager leader election role
*/}}
{{- define "gpuPartitioner.leaderElectionRoleName" -}}
{{ include "gpuPartitioner.fullname" . }}-leader-election
{{- end }}

{{/*
Create the name of the controller manager auth proxy role
*/}}
{{- define "gpuPartitioner.authProxyRoleName" -}}
{{ include "gpuPartitioner.fullname" . }}-auth-proxy
{{- end }}

{{/*
Create the name of the controller manager metrics reader role
*/}}
{{- define "gpuPartitioner.metricsReaderRoleName" -}}
{{ include "gpuPartitioner.fullname" . }}-metrics-reader
{{- end }}

{{/*
*********************************************************************
* MIG Agent
*********************************************************************
*/}}

{{/*
Name of the mig-agent
*/}}
{{- define "migAgent.name" -}}
{{- "nos-mig-agent" -}}
{{- end }}

{{- define "migAgent.fullname" -}}
{{- printf "%s-%s" .Release.Name "mig-agent" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
MIG Agent labels
*/}}
{{- define "migAgent.labels" -}}
helm.sh/chart: {{ include "gpuPartitioner.chart" . }}
{{ include "migAgent.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ "nos" }}
app.kubernetes.io/component: mig-agent
{{- end }}

{{/*
MIG Agent selector labels
*/}}
{{- define "migAgent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "migAgent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the file storing the MIG Agent configuration
*/}}
{{- define "migAgent.configFileName" -}}
mig_agent_config.yaml
{{- end }}


{{/*
Create the name of the MIG Agent config ConfigMap
*/}}
{{- define "migAgent.config.configMapName" -}}
{{- include "migAgent.fullname" . }}-config
{{- end }}

{{/*
*********************************************************************
* GPU Agent
*********************************************************************
*/}}

{{/*
Name of the gpu-agent
*/}}
{{- define "gpuAgent.name" -}}
{{- "nos-gpu-agent" -}}
{{- end }}

{{- define "gpuAgent.fullname" -}}
{{- printf "%s-%s" .Release.Name "gpu-agent" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
GPU Agent labels
*/}}
{{- define "gpuAgent.labels" -}}
helm.sh/chart: {{ include "gpuPartitioner.chart" . }}
{{ include "gpuAgent.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ "nos" }}
app.kubernetes.io/component: gpu-agent
{{- end }}

{{/*
GPU agent selector labels
*/}}
{{- define "gpuAgent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gpuAgent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the file storing the GPU Agent configuration
*/}}
{{- define "gpuAgent.configFileName" -}}
gpu_agent_config.yaml
{{- end }}

{{/*
Create the name of the GPU agent config ConfigMap
*/}}
{{- define "gpuAgent.config.configMapName" -}}
{{- include "gpuAgent.fullname" . }}-config
{{- end }}
