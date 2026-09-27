{{- define "nexwall.fullname" -}}
tenant-{{ .Values.tenantId }}-{{ .Values.slug }}
{{- end -}}

{{/* usage: {{ include "nexwall.image" (dict "root" $ "component" "vpn") }} */}}
{{- define "nexwall.image" -}}
{{ .root.Values.image.registry }}/{{ .root.Values.image.owner }}/nethsecurity-{{ .component }}:{{ .root.Values.image.tag }}
{{- end -}}

{{- define "nexwall.labels" -}}
app.kubernetes.io/part-of: nexwall-controller
tenant-id: "{{ .Values.tenantId }}"
tenant-slug: {{ .Values.slug }}
{{- end -}}
