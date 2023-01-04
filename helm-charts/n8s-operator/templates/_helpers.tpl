{{/*
Create operator name
*/}}
{{- define "operator.name" -}}
{{- printf "%s-%s" .Values.namePrefix "operator" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Operator labels
*/}}
{{- define "operator.labels" -}}
helm.sh/chart: {{ include "operator.chart" . }}
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
app.kubernetes.io/name: operator
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: {{ "nebulnetes" }}
{{- end }}

{{/*
Create the name of the gpu partitioner config ConfigMap
*/}}
{{- define "operator.config.configMapName" -}}
{{- include "operator.name" . }}-config
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
{{ .Values.namePrefix }}-webhook-server-cert
{{- end }}


{{/*
Create the name of the service pointing to the operator webhook server
*/}}
{{- define "operator.webhookServiceName" -}}
{{ .Values.namePrefix }}-webhook
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

{{/*
Create the name of the self-signed certificate issuer
*/}}
{{- define "selfSignedCertIssuerName" -}}
{{ .Values.namePrefix }}-selfsigned-issuer
{{- end }}
