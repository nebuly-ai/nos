{{/*
*********************************************************************
* GPU Partitioner
*********************************************************************
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "gpu-partitioner.name" -}}
{{- .Chart.Name -}}
{{- end }}

{{- define "gpu-partitioner.fullname" -}}
{{- $name := .Chart.Name -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- (printf "%s-%s" .Release.Name $name) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "gpu-partitioner.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
GPU Partitioner labels
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
GPU Partitioner selector labels
*/}}
{{- define "gpu-partitioner.selectorLabels" -}}
app.kubernetes.io/name: gpu-partitioner
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: {{ "nebulnetes" }}
{{- end }}

{{/*
Create the name of the gpu partitioner config ConfigMap
*/}}
{{- define "gpu-partitioner.config.configMapName" -}}
{{- include "gpu-partitioner.fullname" . }}-config
{{- end }}

{{/*
Create the name of the known MIG geometries ConfigMap
*/}}
{{- define "gpu-partitioner.knownMigGeometriesConfigMapName" -}}
{{- include "gpu-partitioner.fullname" . }}-known-mig-geometries
{{- end }}

{{/*
Create the name of the file storing the possible MIG geometries of each known GPU model
*/}}
{{- define "gpu-partitioner.knownMigGeometriesFileName" -}}
known_mig_geometries.yaml
{{- end }}

{{/*
Create the name of the file storing the scheduler config
*/}}
{{- define "gpu-partitioner.schedulerConfigFileName" -}}
scheduler_config.yaml
{{- end }}

{{/*
Create the name of the file storing the GPU Partitioner configuration
*/}}
{{- define "gpu-partitioner.configFileName" -}}
gpu_partitioner_config.yaml
{{- end }}

{{/*
Create the name of the controller manager leader election role
*/}}
{{- define "gpu-partitioner.leaderElectionRoleName" -}}
{{ include "gpu-partitioner.fullname" . }}-leader-election
{{- end }}

{{/*
Create the name of the controller manager auth proxy role
*/}}
{{- define "gpu-partitioner.authProxyRoleName" -}}
{{ include "gpu-partitioner.fullname" . }}-auth-proxy
{{- end }}

{{/*
Create the name of the controller manager metrics reader role
*/}}
{{- define "gpu-partitioner.metricsReaderRoleName" -}}
{{ include "gpu-partitioner.fullname" . }}-metrics-reader
{{- end }}

{{/*
*********************************************************************
* MIG Agent
*********************************************************************
*/}}

{{/*
Name of the mig-agent
*/}}
{{- define "mig-agent.name" -}}
{{- "mig-agent" -}}
{{- end }}

{{- define "mig-agent.fullname" -}}
{{- $name := "mig-agent" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | replace .Chart.Name $name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- (printf "%s-%s" .Release.Name $name) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
MIG Agent labels
*/}}
{{- define "mig-agent.labels" -}}
helm.sh/chart: {{ include "gpu-partitioner.chart" . }}
{{ include "mig-agent.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
MIG Agent selector labels
*/}}
{{- define "mig-agent.selectorLabels" -}}
app.kubernetes.io/name: mig-agent
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: {{ "nebulnetes" }}
{{- end }}

{{/*
Create the name of the file storing the MIG Agent configuration
*/}}
{{- define "mig-agent.configFileName" -}}
mig_agent_config.yaml
{{- end }}


{{/*
Create the name of the MIG Agent config ConfigMap
*/}}
{{- define "mig-agent.config.configMapName" -}}
{{- include "mig-agent.fullname" . }}-config
{{- end }}

{{/*
*********************************************************************
* Time Slicing Agent
*********************************************************************
*/}}

{{/*
Name of the time-slicing-agent
*/}}
{{- define "time-slicing-agent.name" -}}
{{- "time-slicing-agent" -}}
{{- end }}

{{- define "time-slicing-agent.fullname" -}}
{{- $name := "time-slicing-agent" -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- .Release.Name | replace .Chart.Name $name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- (printf "%s-%s" .Release.Name $name) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Time Slicing Agent labels
*/}}
{{- define "time-slicing-agent.labels" -}}
helm.sh/chart: {{ include "gpu-partitioner.chart" . }}
{{ include "time-slicing-agent.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Time Slicing agent selector labels
*/}}
{{- define "time-slicing-agent.selectorLabels" -}}
app.kubernetes.io/name: time-slicing-agent
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: {{ "nebulnetes" }}
{{- end }}

{{/*
Create the name of the file storing the time-slicing Agent configuration
*/}}
{{- define "time-slicing-agent.configFileName" -}}
time_slicing_agent_config.yaml
{{- end }}

{{/*
Create the name of the time-slicing agent config ConfigMap
*/}}
{{- define "time-slicing-agent.config.configMapName" -}}
{{- include "time-slicing-agent.fullname" . }}-config
{{- end }}
