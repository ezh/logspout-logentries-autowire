
package logspout

import (
    "crypto/tls"
    "encoding/json"
    "github.com/fsouza/go-dockerclient"
    "github.com/gliderlabs/logspout/router"
    "io/ioutil"
    "net/http"
    "time"
    "fmt"
    "errors"
)

type LeaObj struct {
  route                    *router.Route
}

var DockerHost string
var InstanceID string
var DefaultToken string

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

  DefaultToken = route.Address

  return &LeaObj{
    route:         route,
  }, nil

}

type LogentriesMessage struct {
  Id         string                 `json:"id,omitempty"`
  Image      string                 `json:"image,omitempty"`
  Created    string                 `json:"created,omitempty"`
  Status     string                 `json:"status,omitempty"`
  Node       string                 `json:"node,omitempty"`
  InstanceId string                 `json:"instanceid,omitempty"`
  Labels     map[string]string      `json:"labels,omitempty"`
  Line       map[string]interface{} `json:"line,omitempty"`
}


func (a *LeaObj) Stream(logstream chan *router.Message) {

  logger := NewLogger()

  for m := range logstream {

    token := m.Container.Config.Labels["logentries.token"]
    if token == "" {
      token = DefaultToken
    }

    containerID := m.Container.Config.Hostname
    imageID := m.Container.Config.Image
    created := m.Container.Created
    labels := m.Container.Config.Labels
    status := m.Container.State.Status

    var logline map[string]interface{}
    err := json.Unmarshal([]byte(m.Data), &logline)

    if err != nil{
      logline = make(map[string]interface{})
      logline["message"] = m.Data
    }

    payload := LogentriesMessage{Id: containerID, Image: imageID, Created: created.String(), Status: status , Node: DockerHost, InstanceId: InstanceID, Labels: labels, Line: logline}
    jsonPayload, jsonerr := json.Marshal(payload)

    if jsonerr != nil {
      panic(jsonerr)
    }

    msg_to_send := []byte(token + " " + string(jsonPayload) + "\n")
    logger.Write(&msg_to_send)

  }

}

type Logger struct {
  conn           *tls.Conn
  messageChannel chan *[]byte
}

func NewLogger() *Logger {
  l := Logger{
    messageChannel: make(chan *[]byte, 1000),
  }
  go l.handleMessages()
  return &l
}

func (l *Logger) handleMessages() {
  const defaultBackoff = 100 * time.Millisecond
  const maxBackoff = 5 * time.Second
  nextBackoff := defaultBackoff

  var conn *tls.Conn
  var err error
  for {
    msg := <-l.messageChannel
    for {
      if conn == nil {
        conn, err = tls.Dial("tcp", "data.logentries.com:443", &tls.Config{})
        if err != nil {
          fmt.Println("open failed, backing off", nextBackoff, err)
          time.Sleep(nextBackoff)
          nextBackoff *= 2
          if nextBackoff > maxBackoff {
            nextBackoff = maxBackoff
          }
          continue
        } else {
          nextBackoff = defaultBackoff
        }
      }
      if _, err := conn.Write(*msg); err != nil {
        conn = nil
        fmt.Println("write failed, backing off", nextBackoff, err)
        time.Sleep(nextBackoff)
        nextBackoff *= 2
        if nextBackoff > maxBackoff {
          nextBackoff = maxBackoff
        }
        continue
      } else {
        nextBackoff = defaultBackoff
      }
      break
    }
  }
}

func (l *Logger) Write(data *[]byte) error {
  select {
  case l.messageChannel <- data:
    return nil
  default:
    return errors.New("message buffer full, dropping message")
  }

}
