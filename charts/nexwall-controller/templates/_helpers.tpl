{{- define "nexwall.fullname" -}}
tenant-{{ .Values.tenantId }}-{{ .Values.slug }}
{{- end -}}

{{/* usage: include "nexwall.image" (dict "root" $ "component" "vpn") */}}
{{- define "nexwall.image" -}}
{{ .root.Values.image.registry }}/{{ .root.Values.image.owner }}/nethsecurity-{{ .component }}:{{ .root.Values.image.tag }}
{{- end -}}

{{- define "nexwall.labels" -}}
app.kubernetes.io/part-of: nexwall-controller
tenant-id: "{{ .Values.tenantId }}"
tenant-slug: {{ .Values.slug }}
{{- end -}}

{{/* vpnCidr is always 10.X.0.0/16 (see values.schema.json) */}}
{{- define "nexwall.vpnPrefix" -}}
{{ .Values.vpnCidr | trimSuffix ".0.0/16" }}
{{- end -}}
{{- define "nexwall.vpnNetwork" -}}
{{ include "nexwall.vpnPrefix" . }}.0.0
{{- end -}}
{{- define "nexwall.vpnServerIP" -}}
{{ include "nexwall.vpnPrefix" . }}.0.1
{{- end -}}

{{- define "nexwall.scheme" -}}
{{ if .Values.ingress.tls.enabled }}https{{ else }}http{{ end }}
{{- end -}}
