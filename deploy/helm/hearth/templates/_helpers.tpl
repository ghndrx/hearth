{{/*
Expand the name of the chart and all resources
*/ -}}
{{- define "hearth.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "hearth.fullname" -}}
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

{{- define "hearth.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "_" "-" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "hearth.labels" -}}
app.kubernetes.io/name: {{ include "hearth.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/component: {{ .Values.backend.fullnameOverride | default "backend" }}
app.kubernetes.io/part-of: {{ include "hearth.name" . }}
heritage: {{ .Release.Service }}
{{- end }}

{{- define "hearth.selectorLabels" -}}
app.kubernetes.io/name: {{ include "hearth.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "hearth.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "hearth.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{ .Values.serviceAccount.name | default "default" }}
{{- end }}
{{- end }}
