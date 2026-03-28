{{/*
Expand the name of the chart.
*/}}
{{- define "fusionauth.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "fusionauth.fullname" -}}
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
{{- define "fusionauth.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "fusionauth.labels" -}}
helm.sh/chart: {{ include "fusionauth.chart" . }}
{{ include "fusionauth.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: hearth
{{- end }}

{{/*
Selector labels
*/}}
{{- define "fusionauth.selectorLabels" -}}
app.kubernetes.io/name: {{ include "fusionauth.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Secret name for FusionAuth credentials
*/}}
{{- define "fusionauth.secretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- include "fusionauth.fullname" . }}-secret
{{- end }}
{{- end }}

{{/*
Root database secret name
*/}}
{{- define "fusionauth.rootSecretName" -}}
{{- if .Values.database.root.existingSecret }}
{{- .Values.database.root.existingSecret }}
{{- else }}
{{- include "fusionauth.fullname" . }}-secret
{{- end }}
{{- end }}
