{{/*
Create operator name
*/}}
{{- define "operator.name" -}}
{{- "operator" -}}
{{- end }}

{{- define "operator.fullname" -}}
{{- $name := include "operator.name" . -}}
{{- if contains .Chart.Name .Release.Name -}}
{{- printf "%s-%s" .Chart.Name (.Release.Name | replace .Chart.Name $name | trunc 63 | trimSuffix "-") -}}
{{- else -}}
{{- (printf "%s-%s" .Release.Name $name) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Operator labels
*/}}
{{- define "operator.labels" -}}
helm.sh/chart: {{ include "n8s.chart" . }}
{{ include "operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
GPU Partitioner selector labels
*/}}
{{- define "operator.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: operator
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
