{{/* vim: set filetype=mustache: */}}
{{/*
Renders a value that contains template.
Usage:
{{ include "common.tplvalues.render" ( dict "value" .Values.path.to.the.Value "context" $) }}

From: https://raw.githubusercontent.com/bitnami/charts/main/bitnami/common/templates/_tplvalues.tpl

See: https://stackoverflow.com/questions/72816925/helm-templating-in-configmap-for-values-yaml
*/}}
{{- define "common.tplvalues.render" -}}
    {{- if typeIs "string" .value }}
        {{- tpl .value .context }}
    {{- else }}
        {{- tpl (.value | toYaml) .context }}
    {{- end }}
{{- end -}}
