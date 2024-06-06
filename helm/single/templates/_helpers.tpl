{{/*
Expand the name of the chart.
*/}}
{{- define "single.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "single.fullname" -}}
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
{{- define "single.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "single.labels" -}}
helm.sh/chart: {{ include "single.chart" . }}
{{ include "single.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "single.selectorLabels" -}}
app.kubernetes.io/name: {{ include "single.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "single.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "single.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "common.env" }}
- name: DOMAIN
  value: {{ .Chart.Name }}
- name: NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: DEPLOYMENT_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.labels['app.kubernetes.io/name']
{{ end -}}

{{- define "otel.env" }}
{{- if .Values.otlp.endpoint }}
- name: OTEL_EXPORTER_OTLP_ENDPOINT
  value: {{ .Values.otlp.endpoint }}
{{- end }}
{{- if .Values.otlp.insecure }}
- name: OTEL_EXPORTER_OTLP_INSECURE
  value: {{ .Values.otlp.insecure | quote }}
{{- end }}
{{- if .Values.otlp.headers }}
- name: OTEL_EXPORTER_OTLP_HEADERS
  value: {{ .Values.otlp.headers | quote }}
{{- end }}
{{ end -}}


{{- define "chaosmania.container" }}
- name: chaosmania
  image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
  imagePullPolicy: {{.Values.image.pullPolicy}}
  args:
    - "server"
    - "--port"
    - "8080"
  ports:
    - name: http
      containerPort: 8080
      protocol: TCP
  livenessProbe:
    httpGet:
      path: /health
      port: http
  readinessProbe:
    httpGet:
      path: /health
      port: http
  startupProbe:
    httpGet:
      path: /health
      port: http
  securityContext: {{- toYaml .Values.securityContext | nindent 4}}
  resources: {{- toYaml .Values.resources | nindent 4}}
  volumeMounts:
    {{ if .Values.persistence.enabled }}
    - mountPath: "/data"
      name: repository
    {{ end }}
  env:
    {{- include "otel.env" . | nindent 4 }}
    {{- include "common.env" . | nindent 4 }}
{{ end -}}

{{- define "chaosmania.service" }}
apiVersion: v1
kind: Service
metadata:
  name: {{ . }}
  labels:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
    scrape-prometheus: "true"
spec:
  type: ClusterIP
  ports:
    - port: 8080
      targetPort: http
      name: http
  selector:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
{{ end -}}
