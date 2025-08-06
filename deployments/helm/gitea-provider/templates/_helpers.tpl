{{/*
Expand the name of the chart.
*/}}
{{- define "gitea-provider.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "gitea-provider.fullname" -}}
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
{{- define "gitea-provider.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "gitea-provider.labels" -}}
helm.sh/chart: {{ include "gitea-provider.chart" . }}
{{ include "gitea-provider.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: crossplane-provider
app.kubernetes.io/part-of: crossplane
crossplane.io/provider: gitea
provider.crossplane.io/architecture: complete-native
{{- end }}

{{/*
Selector labels
*/}}
{{- define "gitea-provider.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gitea-provider.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "gitea-provider.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "gitea-provider.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Provider configuration name
*/}}
{{- define "gitea-provider.configName" -}}
{{- printf "%s-config" (include "gitea-provider.fullname" .) }}
{{- end }}

{{/*
Provider secret name  
*/}}
{{- define "gitea-provider.secretName" -}}
{{- printf "%s-secret" (include "gitea-provider.fullname" .) }}
{{- end }}

{{/*
Create provider image reference
*/}}
{{- define "gitea-provider.image" -}}
{{- printf "%s:%s" .Values.provider.image.repository (.Values.provider.image.tag | default .Chart.AppVersion) }}
{{- end }}

{{/*
Generate monitoring labels
*/}}
{{- define "gitea-provider.monitoringLabels" -}}
{{- if .Values.monitoring.serviceMonitor.labels }}
{{- toYaml .Values.monitoring.serviceMonitor.labels }}
{{- end }}
app.kubernetes.io/name: {{ include "gitea-provider.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Generate security context
*/}}
{{- define "gitea-provider.securityContext" -}}
runAsNonRoot: true
runAsUser: 65534
allowPrivilegeEscalation: false
capabilities:
  drop: ["ALL"]
seccompProfile:
  type: RuntimeDefault
{{- if .Values.provider.securityContext }}
{{- toYaml .Values.provider.securityContext }}
{{- end }}
{{- end }}

{{/*
Generate resource requirements
*/}}
{{- define "gitea-provider.resources" -}}
{{- if .Values.provider.resources }}
{{- toYaml .Values.provider.resources }}
{{- else }}
requests:
  memory: "256Mi"
  cpu: "100m"
limits:
  memory: "512Mi"
  cpu: "500m"
{{- end }}
{{- end }}

{{/*
Generate environment variables
*/}}
{{- define "gitea-provider.env" -}}
- name: POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: POD_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: NODE_NAME
  valueFrom:
    fieldRef:
      fieldPath: spec.nodeName
{{- if .Values.development.debug }}
- name: DEBUG
  value: "true"
- name: LOG_LEVEL
  value: "debug"
{{- else }}
- name: LOG_LEVEL
  value: "info"
{{- end }}
{{- if .Values.monitoring.enabled }}
- name: ENABLE_METRICS
  value: "true"
- name: METRICS_ADDR
  value: "0.0.0.0:8080"
{{- end }}
{{- if .Values.enterprise.auditLogging.enabled }}
- name: ENABLE_AUDIT_LOGGING
  value: "true"
{{- end }}
{{- if .Values.custom.environment }}
- name: ENVIRONMENT
  value: {{ .Values.custom.environment | quote }}
{{- end }}
{{- end }}

{{/*
Generate probe configuration
*/}}
{{- define "gitea-provider.probes" -}}
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
    scheme: HTTP
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  successThreshold: 1
  failureThreshold: 3
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
    scheme: HTTP
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  successThreshold: 1
  failureThreshold: 3
{{- end }}

{{/*
Generate anti-affinity rules for HA
*/}}
{{- define "gitea-provider.antiAffinity" -}}
{{- if gt (int .Values.provider.replicas) 1 }}
podAntiAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
  - weight: 100
    podAffinityTerm:
      labelSelector:
        matchLabels:
          {{- include "gitea-provider.selectorLabels" . | nindent 10 }}
      topologyKey: kubernetes.io/hostname
  - weight: 50
    podAffinityTerm:
      labelSelector:
        matchLabels:
          {{- include "gitea-provider.selectorLabels" . | nindent 10 }}
      topologyKey: topology.kubernetes.io/zone
{{- end }}
{{- if .Values.provider.affinity }}
{{- toYaml .Values.provider.affinity }}
{{- end }}
{{- end }}

{{/*
Generate network policy rules
*/}}
{{- define "gitea-provider.networkPolicyRules" -}}
{{- if .Values.security.networkPolicy.ingress.enabled }}
ingress:
{{- range .Values.security.networkPolicy.ingress.from }}
- from:
  {{- toYaml . | nindent 2 }}
{{- end }}
- from: []
  ports:
  - protocol: TCP
    port: 8080
{{- end }}
{{- if .Values.security.networkPolicy.egress.enabled }}
egress:
{{- range .Values.security.networkPolicy.egress.to }}
- to:
  {{- toYaml . | nindent 2 }}
{{- end }}
- to: []
  ports:
  - protocol: UDP
    port: 53
- to: []
  ports:
  - protocol: TCP
    port: 443
  - protocol: TCP
    port: 80
{{- end }}
{{- end }}

{{/*
Generate backup configuration
*/}}
{{- define "gitea-provider.backupConfig" -}}
{{- if .Values.backup.enabled }}
schedule: {{ .Values.backup.schedule | quote }}
retention:
  {{- toYaml .Values.backup.retention | nindent 2 }}
storage:
  {{- toYaml .Values.backup.storage | nindent 2 }}
destinations:
  {{- toYaml .Values.backup.destinations | nindent 2 }}
{{- end }}
{{- end }}

{{/*
Generate enterprise configuration
*/}}
{{- define "gitea-provider.enterpriseConfig" -}}
{{- if .Values.enterprise.multiTenancy.enabled }}
multiTenancy:
  enabled: true
  isolationLevel: {{ .Values.enterprise.multiTenancy.isolationLevel }}
{{- end }}
{{- if .Values.enterprise.compliance.enabled }}
compliance:
  enabled: true
  standards:
    {{- toYaml .Values.enterprise.compliance.standards | nindent 4 }}
{{- end }}
{{- if .Values.enterprise.auditLogging.enabled }}
auditLogging:
  enabled: true
  destination: {{ .Values.enterprise.auditLogging.destination }}
  retention: {{ .Values.enterprise.auditLogging.retention }}
{{- end }}
{{- if .Values.enterprise.advancedSecurity.enabled }}
advancedSecurity:
  enabled: true
  features:
    {{- toYaml .Values.enterprise.advancedSecurity.features | nindent 4 }}
{{- end }}
{{- end }}