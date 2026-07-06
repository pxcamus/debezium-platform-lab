{{- define "apicurio-registry.fullname" -}}
{{- .Release.Name -}}
{{- end -}}

{{- define "apicurio-registry.uiFullname" -}}
{{- .Release.Name -}}-ui
{{- end -}}