{{- define "common.env" }}
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
      fieldPath: metadata.labels['app']
{{ end -}}

{{- define "common.labels" }}
app.kubernetes.io/instance: {{ . }}
app.kubernetes.io/name: {{ . }}
tags.datadoghq.com/env: test
tags.datadoghq.com/service: {{ . }}
tags.datadoghq.com/version: "1"
scrape-prometheus: "true"
{{ end -}}

{{- define "common.volumes" }}
- name: services
  secret:
    secretName: services
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
  volumeMounts:
{{- if .Values.datadog.enabled }}
    - name: apmsocketpath
      mountPath: /var/run/datadog
{{ end }}
    - mountPath: /etc/chaosmania/ 
      name: services
      readOnly: true
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
    {{- include "datadog.env" . | nindent 4 }}
{{ end -}}

{{- define "chaosmania.service" }}
---
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

{{- define "redis.service" }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ . }}
  labels:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
spec:
  type: ClusterIP
  ports:
    - port: 6379
      name: redis
      targetPort: redis
  selector:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
{{ end -}}

{{- define "redis.statefulset" }}
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ . }}
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ . }}
      app.kubernetes.io/name: {{ . }}
  serviceName: "{{ . }}"
  replicas: 1
  template:
    metadata:
      labels:
        app.kubernetes.io/instance: {{ . }}
        app.kubernetes.io/name: {{ . }}
    spec:
      terminationGracePeriodSeconds: 10
      containers:
        - name: redis
          image: redis:7.0-bullseye
          ports:
            - containerPort: 6379
              name: redis
          volumeMounts:
            - name: data
              mountPath: /data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
{{ end -}}

{{- define "postgres.service" }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ . }}
  labels:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
spec:
  type: ClusterIP
  ports:
    - port: 5432
      name: postgres
      targetPort: postgres
  selector:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
{{ end -}}

{{- define "postgres.statefulset" }}
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ . }}
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ . }}
      app.kubernetes.io/name: {{ . }}
  serviceName: "{{ . }}"
  replicas: 1
  template:
    metadata:
      labels:
        app.kubernetes.io/instance: {{ . }}
        app.kubernetes.io/name: {{ . }}
    spec:
      terminationGracePeriodSeconds: 10
      containers:
        - name: postgres
          image: postgres:14-bullseye
          env:
            - name: POSTGRES_PASSWORD
              value: postgres
          ports:
            - containerPort: 5432
              name: postgres
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
{{ end -}}

{{- define "kafka.statefulset" }}
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ . }}
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ . }}
      app.kubernetes.io/name: {{ . }}
  serviceName: "{{ . }}"
  replicas: 1
  template:
    metadata:
      labels:
        app.kubernetes.io/instance: {{ . }}
        app.kubernetes.io/name: {{ . }}
    spec:
      securityContext:
        fsGroup: 1001
      terminationGracePeriodSeconds: 10
      containers:
        - name: kafka
          image: docker.io/bitnami/kafka:3.6
          env:
            - name: KAFKA_CFG_NODE_ID
              value: "0"
            - name: KAFKA_CFG_PROCESS_ROLES
              value: controller,broker
            - name: KAFKA_CFG_CONTROLLER_QUORUM_VOTERS
              value: 0@{{ . }}:9093
            - name: KAFKA_CFG_LISTENERS
              value: PLAINTEXT://:9092,CONTROLLER://:9093
            - name: KAFKA_CFG_ADVERTISED_LISTENERS
              value: PLAINTEXT://:9092
            - name: KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP
              value: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
            - name: KAFKA_CFG_CONTROLLER_LISTENER_NAMES
              value: CONTROLLER
            - name: KAFKA_CFG_INTER_BROKER_LISTENER_NAME
              value: PLAINTEXT
            - name: KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE
              value: "true"
          ports:
            - containerPort: 9092
              name: kafka
            - containerPort: 9093
              name: kafka2
          volumeMounts:
            - name: data
              mountPath: /bitnami/kafka
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
{{ end -}}

{{- define "kafka.service" }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ . }}
  labels:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
spec:
  type: ClusterIP
  ports:
    - port: 9092
      name: kafka
      targetPort: kafka
    - port: 9093
      name: kafka2
      targetPort: kafka2
  selector:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
{{ end -}}

{{- define "rabbitmq.statefulset" }}
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ . }}
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ . }}
      app.kubernetes.io/name: {{ . }}
  serviceName: "{{ . }}"
  replicas: 1
  template:
    metadata:
      labels:
        app.kubernetes.io/instance: {{ . }}
        app.kubernetes.io/name: {{ . }}
    spec:
      terminationGracePeriodSeconds: 10
      containers:
        - name: rabbitmq
          image: rabbitmq:3-management
          ports:
            - containerPort: 15672
              name: rabbit1
            - containerPort: 5672
              name: rabbit2
{{ end -}}


{{- define "rabbitmq.service" }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ . }}
  labels:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
spec:
  type: ClusterIP
  ports:
    - port: 15672
      name: rabbit1 
      targetPort: rabbit1
    - port: 5672
      name: rabbit2
      targetPort: rabbit2
  selector:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
{{ end -}}

{{- define "minio.statefulset" }}
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ . }}
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ . }}
      app.kubernetes.io/name: {{ . }}
  serviceName: "{{ . }}"
  replicas: 1
  template:
    metadata:
      labels:
        app.kubernetes.io/instance: {{ . }}
        app.kubernetes.io/name: {{ . }}
    spec:
      terminationGracePeriodSeconds: 10
      containers:
        - name: minio
          image: minio/minio
          args:
            - server
            - /data
            - --console-address
            - :9001
            - --address 
            - :9000
          ports:
            - containerPort: 9000
              name: minio1
            - containerPort: 9001
              name: minio2
{{ end -}}


{{- define "minio.service" }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ . }}
  labels:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
spec:
  type: ClusterIP
  ports:
    - port: 9000
      name: minio1
      targetPort: minio1
    - port: 9001
      name: minio2
      targetPort: minio2
  selector:
    app.kubernetes.io/instance: {{ . }}
    app.kubernetes.io/name: {{ . }}
{{ end -}}


{{- define "chaosmania.hpa" }}
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ . }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ . }}
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 80
{{ end -}}
