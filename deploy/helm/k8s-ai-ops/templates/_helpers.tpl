{{- define "k8s-ai-ops.image" -}}
{{- $root := index . 0 -}}
{{- $repo := index . 1 -}}
{{- if $root.Values.images.registry -}}
{{ printf "%s/%s:%s" $root.Values.images.registry $repo $root.Values.images.tag }}
{{- else -}}
{{ printf "%s:%s" $repo $root.Values.images.tag }}
{{- end -}}
{{- end -}}
