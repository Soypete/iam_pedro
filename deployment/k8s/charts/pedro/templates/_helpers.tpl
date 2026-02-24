{{/*
Common labels
*/}}
{{- define "pedro.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: pedro
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end }}

{{/*
Discord labels
*/}}
{{- define "pedro.discord.labels" -}}
{{ include "pedro.labels" . }}
app.kubernetes.io/name: pedro-discord
app.kubernetes.io/component: discord
{{- end }}

{{/*
Discord selector labels
*/}}
{{- define "pedro.discord.selectorLabels" -}}
app.kubernetes.io/name: pedro-discord
app.kubernetes.io/component: discord
{{- end }}

{{/*
Twitch labels
*/}}
{{- define "pedro.twitch.labels" -}}
{{ include "pedro.labels" . }}
app.kubernetes.io/name: pedro-twitch
app.kubernetes.io/component: twitch
{{- end }}

{{/*
Twitch selector labels
*/}}
{{- define "pedro.twitch.selectorLabels" -}}
app.kubernetes.io/name: pedro-twitch
app.kubernetes.io/component: twitch
{{- end }}

{{/*
Keepalive labels
*/}}
{{- define "pedro.keepalive.labels" -}}
{{ include "pedro.labels" . }}
app.kubernetes.io/name: pedro-keepalive
app.kubernetes.io/component: keepalive
{{- end }}

{{/*
Image pull secrets
*/}}
{{- define "pedro.imagePullSecrets" -}}
{{- if .Values.global.imagePullSecrets }}
imagePullSecrets:
{{- range .Values.global.imagePullSecrets }}
  - name: {{ .name }}
{{- end }}
{{- end }}
{{- end }}
