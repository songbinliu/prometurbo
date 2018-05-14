package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"html/template"
	"io"
	"net/http"

	"github.com/turbonomic/prometurbo/appmetric/pkg/inter"
	"github.com/turbonomic/prometurbo/appmetric/pkg/util"
)

var (
	htmlHeadTemplate string = `
	<html><head><title>{{.PageTitle}}</title>
	<link rel="icon" type="image/jpg" href="data:;base64,iVBORw0KGgo">
	</head><boday><center>
	<h1>{{.PageHead}}</h1>
	This is a web server to serve Application latency and request-count-per-second.
	<hr width="50%">
	`

	htmlWelcomeTemplate string = `

	<p>
	<table style="font-size:18px">
	<tr><td><a href="/index.html"> welcome Page </a></td><td> this page </td></tr>
	<tr><td><a href="{{.PodPath}}"> Pod metrics </a></td><td> response-time: ms, request-count</td></tr>
	<tr><td><a href="{{.ServicePath}}"> Service metrics </a></td><td> response-time: ms, request-count</td></tr>
	</table>
	</p>

	Incoming path is: {{.IncomePath}}
	`

	htmlFootTemplate string = `
	<hr width="50%">hostName:  {{.HostName}}
	<br/>
	hostIP: {{.HostIP}} : {{.HostPort}}
	<br/>
	ClientIP: {{.ClientIP}}
	<br/>
	OriginalClient: {{.OriginalClient}}
	</center></body></html>
	`
)

func getHead(title string, head string) (string, error) {
	tmp, err := template.New("head").Parse(htmlHeadTemplate)
	if err != nil {
		glog.Errorf("Failed to parse image template %v:%v", htmlHeadTemplate, err)
		return "", fmt.Errorf("parse failed")
	}

	var result bytes.Buffer
	data := map[string]interface{}{"PageTitle": title, "PageHead": head}
	if err := tmp.Execute(&result, data); err != nil {
		glog.Errorf("Faile to execute template: %v", err)
		return "", fmt.Errorf("execute failed.")
	}

	return result.String(), nil
}

func genWelcomePage(path string) (string, error) {
	//1. get body
	tmp, err := template.New("body").Parse(htmlWelcomeTemplate)
	if err != nil {
		glog.Errorf("Failed to parse image template %v:%v", htmlWelcomeTemplate, err)
		return "", err
	}

	var body bytes.Buffer
	data := map[string]string{"IncomePath": path, "PodPath": appMetricPath, "ServicePath": serviceMetricPath}
	if err = tmp.Execute(&body, data); err != nil {
		glog.Errorf("Failed to execute template: %v", err)
		return "", err
	}

	return body.String(), nil
}

// handle pages "/", "/index.html", "index.htm"
func (s *MetricServer) handleWelcome(path string, w http.ResponseWriter, r *http.Request) {
	//1. head
	head, err := getHead("Welcome", "Introduction")
	if err != nil {
		glog.Errorf("Failed to generate html head.")
		head = "empty head"
	}

	//2. body
	body, err := genWelcomePage(path)
	if err != nil {
		glog.Errorf("Failed to generate html body.")
		body = "empty body"
	}

	//3. foot
	foot := s.genPageFoot(r)

	io.WriteString(w, head+body+foot)
	return
}

func (s *MetricServer) genPageFoot(r *http.Request) string {
	tmp, err := template.New("foot").Parse(htmlFootTemplate)
	if err != nil {
		glog.Errorf("Failed to parse image template %v:%v", htmlFootTemplate, err)
		return ""
	}

	var result bytes.Buffer

	data := make(map[string]interface{})
	data["HostName"] = s.host
	data["HostIP"] = s.ip
	data["HostPort"] = s.port
	data["ClientIP"] = util.GetClientIP(r)
	data["OriginalClient"] = util.GetOriginalClientInfo(r)

	if err := tmp.Execute(&result, data); err != nil {
		glog.Errorf("Faile to execute template: %v", err)
		return ""
	}

	return result.String()
}

func (s *MetricServer) faviconHandler(w http.ResponseWriter, r *http.Request) {
	fpath := "/tmp/favicon.jpg"
	if !util.FileExists(fpath) {
		glog.Warningf("favicon file[%v] does not exist.", fpath)
		return
	}

	http.ServeFile(w, r, fpath)
	return
}

func (s *MetricServer) sendFailure(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)
	w.Write([]byte(`{"status":"error"}`))
	return
}

func (s *MetricServer) sendMetrics(metrics []*inter.EntityMetric, w http.ResponseWriter, r *http.Request) {
	//2. put metrics to response
	resp := inter.NewMetricResponse()
	resp.SetStatus(0, "Success")
	resp.SetMetrics(metrics)

	//3. marshal to json
	result, err := json.Marshal(resp)
	if err != nil {
		glog.Errorf("Failed to marshal json: %v", err)
		s.sendFailure(w, r)
		return
	}

	glog.V(3).Infof("content: %v", string(result))

	//4. send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(result)
	return
}

func (s *MetricServer) handleAppMetric(w http.ResponseWriter, r *http.Request) {
	//1. get metrics
	metrics, err := s.appClient.GetEntityMetrics()
	if err != nil {
		glog.Errorf("Failed to get Application Metrics: %v", err)
		s.sendFailure(w, r)
		return
	}

	glog.V(3).Infof("App metrics num: %v", len(metrics))

	//2. put metrics to response
	s.sendMetrics(metrics, w, r)
	return
}

func (s *MetricServer) handleServiceMetric(w http.ResponseWriter, r *http.Request) {
	//1. get metrics
	metrics, err := s.vappClient.GetEntityMetrics()
	if err != nil {
		glog.Errorf("Failed to get Application Metrics: %v", err)
		s.sendFailure(w, r)
		return
	}

	//2. put metrics to response
	s.sendMetrics(metrics, w, r)
}

func (s *MetricServer) handleFakeMetric(w http.ResponseWriter, r *http.Request) {
	//1. generate fake app metrics
	metrics := inter.GenerateFakeMetrics()
	//2. put metrics to response
	s.sendMetrics(metrics, w, r)
	glog.V(3).Infof("fake metric service finish: %d", len(metrics))
}
