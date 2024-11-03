{{- define "common.env" }}
- name: DOMAIN
  value: {{ .Chart.Name }}
- name: HOST_IP
  valueFrom:
    fieldRef:
      fieldPath: status.hostIP
- name: NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: DEPLOYMENT_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.labels['app.kubernetes.io/name']
{{ end -}}

{{- define "common.labels" }}
app.kubernetes.io/instance: {{ . }}
app.kubernetes.io/name: {{ . }}
app.kubernetes.io/part-of: trading
tags.datadoghq.com/env: test
tags.datadoghq.com/service: {{ . }}
tags.datadoghq.com/version: "1"
{{ end -}}

{{- define "common.volumes" }}
{{- if .Values.datadog.enabled }}
- hostPath:
    path: /var/run/datadog/
  name: apmsocketpath
{{ end -}}
{{ end -}}

{{- define "datadog.env" }}
{{- if .Values.datadog.enabled }}
- name: DATADOG_ENABLED
  value: "true"
- name: DD_TRACE_AGENT_URL
  value: 'unix:///var/run/datadog/apm.socket'
- name: DD_ENV
  valueFrom:
    fieldRef:
      fieldPath: metadata.labels['tags.datadoghq.com/env']
- name: DD_SERVICE
  valueFrom:
    fieldRef:
      fieldPath: metadata.labels['tags.datadoghq.com/service']
- name: DD_VERSION
  valueFrom:
    fieldRef:
      fieldPath: metadata.labels['tags.datadoghq.com/version']
{{ end -}}
{{ end -}}
 
{{- define "otel.env" }}
{{- if .Values.otlp.enabled }}
- name: OTEL_ENABLED
  value: "true"
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
{{- if .Values.datadog.enabled }}
  volumeMounts:
    - name: apmsocketpath
      mountPath: /var/run/datadog
{{ end }}
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    capabilities:
      drop:
      - all
  resources: {{- toYaml .Values.resources | nindent 4}}
  env:
    {{- include "otel.env" . | nindent 4 }}
    {{- include "common.env" . | nindent 4 }}
    {{- include "datadog.env" . | nindent 4 }}
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
