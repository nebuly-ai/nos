{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "nos.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Define nebulnetes full name including the Chart release name
*/}}
{{- define "nos.fullname" -}}
{{- $name := .Chart.Name -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- (printf "%s-%s" .Release.Name $name) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

