package telemetry

import (
	"encoding/base64"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/state/accesspolicy"
	"github.com/KnowitSolutions/istio-oidc/state/session"
	"html/template"
	"net/http"
	"net/url"
)

func RegisterDashboard(mux *http.ServeMux, apStore accesspolicy.Store, sessStore session.Store, ) {
	dash := dashboard{apStore, sessStore}
	mux.Handle("/", &dash)
}

var tmpl = template.Must(template.
	New("dashboard").
	Funcs(map[string]interface{}{
		"fmtUrl": fmtUrl,
		"fmtId":  fmtId,
	}).
	Parse(`
{{define "root"}}
<!DOCTYPE html>
<html lang="en">
	<head><title>Istio OIDC</title></head>
	<style>
	table {
	  border-collapse: collapse;
	}

	th, td {
	  border: 1px solid #999;
	  padding: 0.5rem;
	}
	</style>
<body>
	<h2>Access policies</h2>
	{{range .AccessPolicies}}
	{{template "access-policy" .}}
	{{end}}
	<h2>Sessions</h2>
	{{template "sessions" .}}
	<h2>Links</h2>
	{{template "links"}}
</body>
</html>
{{end}}

{{define "access-policy"}}
<h3>{{.Name}}</h3>
<table>
	<tr><th>Provider</th><td>{{.Oidc.Provider.Name}}</td></tr>
	<tr><th>Callback</th><td>{{.Oidc.Callback | fmtUrl}}</td></tr>
	<tr>
		<th>Virtual hosts</th>
		<td>{{range $i, $v := .VirtualHosts}}{{if $i}}, {{end}}{{$v}}{{end}}</td>
	</tr>
</table>
<h4>Routes</h4>
<table>
	<tr><th>Name</th><th>Enabled</th><th>Roles</th><th>Headers</th></tr>
	<tr><td><em>Default</em></td>{{template "route" .Default}}</tr>
	{{range $k, $v := .Routes}}
	<tr><td>{{$k}}</td>{{template "route" $v}}</tr>
	{{end}}
</table>
{{end}}

{{define "route"}}
<td>{{if .EnableAuthz}}yes{{else}}no{{end}}</td>
<td>{{range $i, $v := .Roles}}{{if $i}}, {{end}}{{$v}}{{end}}</td>
<td>
	{{if .Headers}}
	<table>
		<tr><th>Name</th><th>Value</th><th>Roles</th></tr>
		{{range .Headers}}
		<tr>
			<td>{{.Name}}</td>
			<td>{{.Value}}</td>
			<td>{{range $i, $v := .Roles}}{{if $i}}, {{end}}{{$v}}{{end}}</td>
		</tr>
		{{end}}
	</table>
	{{end}}
</td>
{{end}}

{{define "sessions"}}
<table>
	<tr><th>ID</th><th>Peer</th><th>Expiry</th></tr>
	{{range .Sessions}}
	<tr><td>{{.Id | fmtId}}</td><td>{{.PeerId}}</td><td>{{.Expiry}}</td></tr>
	{{end}}
</table>
{{end}}

{{define "links"}}
<a href="/health">Liveliness probe</a>
<a href="/ready">Readiness probe</a>
<a href="/metrics">Metrics</a>
{{end}}
	`))

type dashboard struct {
	accessPolicies accesspolicy.Store
	sessions       session.Store
}

type dashboardData struct {
	AccessPolicies <-chan accesspolicy.AccessPolicy
	Sessions       <-chan session.Stamped
}

func (r dashboard) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	data := dashboardData{
		r.accessPolicies.Stream(),
		r.sessions.Stream(map[string]uint64{}),
	}
	err := tmpl.ExecuteTemplate(writer, "root", data)
	if err != nil {
		log.Error(req.Context(), err, "Failed rendering dashboard")
	}
}

func fmtUrl(url url.URL) string {
	return url.String()
}

func fmtId(id string) string {
	return base64.StdEncoding.EncodeToString([]byte(id))
}
