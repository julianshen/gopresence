{{/*
Expand the name of the chart.
*/}}
{{- define "presence-service.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "presence-service.fullname" -}}
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
{{- define "presence-service.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "presence-service.labels" -}}
helm.sh/chart: {{ include "presence-service.chart" . }}
{{ include "presence-service.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: {{ .Values.service.nodeType }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "presence-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "presence-service.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the config map
*/}}
{{- define "presence-service.configMapName" -}}
{{- if .Values.configMap.name }}
{{- .Values.configMap.name }}
{{- else }}
{{- include "presence-service.fullname" . }}-config
{{- end }}
{{- end }}

{{/*
Create the name of the secret
*/}}
{{- define "presence-service.secretName" -}}
{{- if .Values.secret.name }}
{{- .Values.secret.name }}
{{- else }}
{{- include "presence-service.fullname" . }}-secret
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "presence-service.image" -}}
{{- $registry := .Values.global.imageRegistry | default "" }}
{{- $repository := .Values.image.repository }}
{{- $tag := .Values.image.tag | toString }}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- else }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}
{{- end }}

{{/*
Generate environment variables for the service
*/}}
{{- define "presence-service.env" -}}
- name: SERVICE_NAME
  value: {{ .Values.service.name | quote }}
- name: SERVICE_VERSION
  value: {{ .Values.service.version | quote }}
- name: SERVICE_PORT
  value: {{ .Values.networking.servicePort | quote }}
- name: NODE_TYPE
  value: {{ .Values.service.nodeType | quote }}
- name: NODE_ID
  value: {{ .Values.service.nodeId | quote }}
- name: NATS_EMBEDDED
  value: {{ .Values.nats.embedded | quote }}
- name: NATS_SERVER_URL
  value: ""
- name: NATS_DATA_DIR
  value: {{ .Values.nats.dataDir | quote }}
- name: NATS_JETSTREAM_MAX_MEMORY
  value: {{ .Values.nats.jetstream.maxMemory | quote }}
- name: NATS_JETSTREAM_MAX_STORE
  value: {{ .Values.nats.jetstream.maxStore | quote }}
- name: NATS_KV_BUCKET
  value: {{ .Values.nats.kv.bucket | quote }}
- name: NATS_KV_TTL
  value: {{ .Values.nats.kv.ttl | quote }}
{{- if .Values.nats.centerUrl }}
- name: NATS_CENTER_URL
  value: {{ .Values.nats.centerUrl | quote }}
{{- end }}
{{- if eq .Values.service.nodeType "center" }}
- name: NATS_LEAF_PORT
  value: {{ .Values.networking.natsLeafPort | quote }}
- name: NATS_CLUSTER_PORT
  value: {{ .Values.networking.natsClusterPort | quote }}
{{- end }}
- name: CACHE_TYPE
  value: {{ .Values.cache.type | quote }}
- name: CACHE_MAX_COST
  value: {{ .Values.cache.maxCost | quote }}
- name: CACHE_NUM_COUNTERS
  value: {{ .Values.cache.numCounters | quote }}
- name: CACHE_BUFFER_ITEMS
  value: {{ .Values.cache.bufferItems | quote }}
- name: CACHE_METRICS
  value: {{ .Values.cache.metrics | quote }}
- name: JWT_SECRET
  valueFrom:
    secretKeyRef:
      name: {{ include "presence-service.secretName" . }}
      key: jwt-secret
- name: JWT_ISSUER
  value: {{ .Values.auth.jwtIssuer | quote }}
- name: JWT_TTL
  value: {{ .Values.auth.jwtTTL | quote }}
- name: LOG_LEVEL
  value: {{ .Values.logging.level | quote }}
- name: LOG_FORMAT
  value: {{ .Values.logging.format | quote }}
{{- end }}