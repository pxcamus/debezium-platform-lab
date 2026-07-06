{{- define "cdc-dashboard.name" -}}
cdc-dashboard
{{- end }}

{{- define "cdc-dashboard.fullname" -}}
{{ .Release.Name }}-cdc-dashboard
{{- end }}