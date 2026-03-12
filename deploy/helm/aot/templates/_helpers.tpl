{{/*
Chart name.
*/}}
{{- define "aot.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name.
*/}}
{{- define "aot.fullname" -}}
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
Common labels.
*/}}
{{- define "aot.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}

{{/*
Selector labels for a component.
*/}}
{{- define "aot.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aot.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Controlplane image reference.
*/}}
{{- define "aot.controlplaneImage" -}}
{{ .Values.images.controlplane.repository }}:{{ .Values.images.controlplane.tag | default .Chart.AppVersion }}
{{- end }}

{{/*
Web image reference.
*/}}
{{- define "aot.webImage" -}}
{{ .Values.images.web.repository }}:{{ .Values.images.web.tag | default .Chart.AppVersion }}
{{- end }}

{{/*
Validate required values.
*/}}
{{- define "aot.validateValues" -}}
{{- if not .Values.temporal.host }}
{{- fail "temporal.host is required. Set it via --set temporal.host=<address:port>" }}
{{- end }}
{{- end }}
