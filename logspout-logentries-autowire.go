
package logspout

import (
//    "errors"
    "encoding/json"
    "github.com/fsouza/go-dockerclient"
    "github.com/gliderlabs/logspout/router"
    "github.com/bsphere/le_go" 
    // "fmt"
    "io/ioutil"
    "net/http"
    "time"
)

type LeaObj struct {
  route                    *router.Route
}

var DockerHost string
var InstanceID string

func init() {
  router.AdapterFactories.Register(LogentriesAutowire, "logentriesautowire")

  endpoint := "unix:///var/run/docker.sock"
  client, _ := docker.NewClient(endpoint)
  DockerInfo, _ := client.Info()
  DockerHost = DockerInfo.Name

  var netClient = &http.Client{
    Timeout: time.Second * 5,
  }

  resp, err := netClient.Get("http://instance-data/latest/meta-data/instance-id")
  if resp != nil {
    defer resp.Body.Close()
  }

  if err != nil {
    InstanceID = "Not AWS"
  } else {
    temp, _ := ioutil.ReadAll(resp.Body)
    InstanceID = string(temp)
  }

}

func LogentriesAutowire (route *router.Route) (router.LogAdapter, error)  {

  return &LeaObj{
    route:         route,
  }, nil

}

type LogentriesMessage struct {
  Id         string                 `json:"id,omitempty"`
  Image      string                 `json:"image,omitempty"`
  Created    string                 `json:"created,omitempty"`
  Status     string                 `json:"status,omitempty"`
  Node       string                  `json:"node,omitempty"`
  InstanceId string                 `json:"instanceid,omitempty"`
  Labels     map[string]string      `json:"labels,omitempty"`
  Line       string                 `json:"line,omitempty"`
}


func (a *LeaObj) Stream(logstream chan *router.Message) {

  connMap := make(map[string]*le_go.Logger)

  for m := range logstream {

    token := m.Container.Config.Labels["logentries.token"]
    if token == "" {
      continue
    }

    containerID := m.Container.Config.Hostname
    imageID := m.Container.Config.Image
    created := m.Container.Created
    labels := m.Container.Config.Labels
    status := m.Container.State.Status

    payload := LogentriesMessage{Id: containerID, Image: imageID, Created: created.String(), Status: status , Node: DockerHost, InstanceId: InstanceID, Labels: labels, Line: m.Data}
    jsonPayload, jsonerr := json.Marshal(payload)

    if jsonerr != nil {
      panic(jsonerr)
    }

    if connMap[token] == nil{
      var err error
      connMap[token], err = le_go.Connect(token)
        if err != nil {
          panic(err)
        }
    }

    connMap[token].Println(string(jsonPayload))

  }

}
