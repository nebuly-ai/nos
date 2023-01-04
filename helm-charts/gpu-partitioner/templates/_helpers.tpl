{{/*
*********************************************************************
* GPU Partitioner
*********************************************************************
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "gpu-partitioner.name" -}}
{{- printf "%s-%s" .Values.namePrefix "gpu-partitioner" }}
{{- end }}

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
*********************************************************************
* MIG Agent
*********************************************************************
*/}}

{{/*
Name of the mig-agent
*/}}
{{- define "mig-agent.name" -}}
{{- printf "%s-%s" .Values.namePrefix "mig-agent" | trunc 63 }}
{{- end }}

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
{{- include "mig-agent.name" . }}-config
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
{{- printf "%s-%s" .Values.namePrefix "time-slicing-agent" | trunc 63 }}
{{- end }}

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
{{- include "time-slicing-agent.name" . }}-config
{{- end }}
{{/*


*********************************************************************
* Misc
*********************************************************************
*/}}

{{/*
Create the name of the controller manager leader election role
*/}}
{{- define "leaderElectionRoleName" -}}
{{ .Values.namePrefix }}-leader-election
{{- end }}

{{/*
Create the name of the controller manager auth proxy role
*/}}
{{- define "authProxyRoleName" -}}
{{ .Values.namePrefix }}-auth-proxy
{{- end }}

{{/*
Create the name of the controller manager metrics reader role
*/}}
{{- define "metricsReaderRoleName" -}}
{{ .Values.namePrefix }}-metrics-reader
{{- end }}
