{{- define "common.env" }}
- name: NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: DEPLOYMENT_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.labels['app']
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
      add: ['NET_BIND_SERVICE']
  resources: {{- toYaml .Values.resources | nindent 4}}
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
