{{/*
Expand the name of the chart.
*/}}
{{- define "operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "operator.labels" -}}
helm.sh/chart: {{ include "operator.chart" . }}
{{ include "operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: nebulnetes
app.kubernetes.io/component: operator
{{- end }}

{{/*
Selector labels
*/}}
{{- define "operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the gpu partitioner config ConfigMap
*/}}
{{- define "operator.config.configMapName" -}}
{{- include "operator.fullname" . }}-config
{{- end }}

{{/*
Create the name of the file storing the scheduler config
*/}}
{{- define "operator.schedulerConfigFileName" -}}
scheduler_config.yaml
{{- end }}

{{/*
Create the name of the file storing the GPU Partitioner configuration
*/}}
{{- define "operator.configFileName" -}}
gpu_partitioner_config.yaml
{{- end }}

{{/*
Create the name of the secret containing the cert of the webhook used for validating CRDs
*/}}
{{- define "operator.webhookCertSecretName" -}}
{{ include "operator.fullname" . }}-webhook-server-cert
{{- end }}

{{/*
Create the name of the service pointing to the operator webhook server
*/}}
{{- define "operator.webhookServiceName" -}}
{{ include "operator.fullname" . }}-webhook
{{- end }}

{{/*
Create the name of the controller manager leader election role
*/}}
{{- define "operator.leaderElectionRoleName" -}}
{{ include "operator.fullname" . }}-leader-election
{{- end }}

{{/*
Create the name of the controller manager auth proxy role
*/}}
{{- define "operator.authProxyRoleName" -}}
{{ include "operator.fullname" . }}-auth-proxy
{{- end }}

{{/*
Create the name of the controller manager metrics reader role
*/}}
{{- define "operator.metricsReaderRoleName" -}}
{{ include "operator.fullname" . }}-metrics-reader
{{- end }}

{{/*
Create the name of the self-signed certificate issuer
*/}}
{{- define "operator.selfSignedCertIssuerName" -}}
{{ include "operator.fullname" . }}-self-signed-issuer
{{- end }}
