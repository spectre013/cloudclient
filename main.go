package cloudclient

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

var logger = logrus.New()

type Result struct {
	Name            string
	Profiles        []string
	Label           string
	Version         string
	State           string
	PropertySources []PropertySources
}

type PropertySources struct {
	Name   string
	Source map[string]interface{}
}

type Property struct {
	sync.Mutex
	Server     string
	Name       string
	Profile    string
	Updated    bool
	Properties map[string]string
}

func Client(server string, appName string, profile string) *Property {
	logger.Out = os.Stdout
	logger.SetLevel(logrus.DebugLevel)

	p := Property{
		Name:       appName,
		Server:     server,
		Profile:    setProfile(profile),
		Updated:    false,
		Properties: make(map[string]string),
	}

	p.FetchProperties()
	go p.Refresh()
	return &p
}

func (p *Property) Refresh() {
	for {
		time.Sleep(time.Second * 30)
		p.FetchProperties()
	}
}

func (p *Property) HasUpdate() bool {
	return p.Updated
}

func (p *Property) GetProperty() *Property {
	return p
}

func (p *Property) GetProperties() map[string]string {
	return p.Properties
}

func (p *Property) FetchProperties() {
	httpAction := createHTTPAction(p.Server, p.Name, p.Profile)

	logger.Info("Fetching cloud configuration from  ", p.Server+"/"+p.Name+p.Profile)
	result, err := DoHttpRequest(httpAction)
	if err != nil {
		logger.Error(err)
	}
	r := toResult(result)
	p.SetValues(r)
}

func (p *Property) SetValues(r *Result) {
	p.Lock()
	cp := p.copy()
	for i := len(r.PropertySources) - 1; i >= 0; i-- {
		props := r.PropertySources[i].Source
		for k, v := range props {
			if val, ok := p.Properties[k]; ok {
				if val != fmt.Sprint(v) {
					p.Properties[k] = fmt.Sprint(v)
				}
			} else {
				p.Properties[k] = fmt.Sprint(v)
			}
		}
	}
	p.PropertyReplacement()
	// only perform this if the current Updated state is false to prevent this from happening before the app
	// can check it resulting in a race condition where this happens before the main app detects changes resulting
	// in the main app not processing the updates
	logger.Info("SetValues: ", p.Updated)
	p.Unlock()
	if !p.Updated {
		p.isUpdated(cp)
	}
}

func (p *Property) isUpdated(cp map[string]string) {
	eq := reflect.DeepEqual(cp, p.Properties)
	if eq {
		p.Updated = false
	} else {
		p.Updated = true
	}
	logger.Info("IsUpdated: ", p.Updated)
}

func (p *Property) copy() map[string]string {
	cp := make(map[string]string)
	for key, value := range p.Properties {
		cp[key] = value
	}
	return cp
}
func (p *Property) PropertyReplacement() {
	for k, v := range p.Properties {
		if strings.Contains(v, "${") {
			name := strings.Replace(v, "${", "", -1)
			name = strings.Replace(name, "}", "", -1)
			p.Properties[k] = p.Properties[name]
		}
	}
}

func setProfile(profile string) string {
	if profile != "" {
		profile = "/" + profile
	} else {
		profile = "/prod"
	}
	return profile
}

func toResult(result []byte) *Result {
	r := new(Result)
	err := json.Unmarshal(result, &r)
	if err != nil {
		logger.Error(err)
	}
	return r
}

func toJson(r *Result) string {
	f, err := json.Marshal(r)
	if err != nil {
		logger.Error("error:", err)
	}
	return string(f)
}

func createHTTPAction(server string, appName string, profile string) HttpAction {
	return HttpAction{
		Url:    server + "/" + appName + profile,
		Method: http.MethodGet,
	}
}
