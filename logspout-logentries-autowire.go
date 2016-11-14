package logspout

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"
)

type logentriesAdaptor struct {
	defaultToken string
	host         string
	instanceID   string
	route        *router.Route
}

func init() {
	router.AdapterFactories.Register(LogentriesAutowire, "logentriesautowire")
}

func LogentriesAutowire(route *router.Route) (router.LogAdapter, error) {

	client, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		return nil, err
	}

	dockerInfo, err := client.Info()
	if err != nil {
		return nil, err
	}

	var httpClient = &http.Client{
		Timeout: time.Second * 5,
	}

	resp, err := httpClient.Get("http://instance-data/latest/meta-data/instance-id")
	if resp != nil {
		defer resp.Body.Close()
	}

	var instanceID string
	if err != nil {
		instanceID = "Not AWS"
	} else {
		temp, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		instanceID = string(temp)
	}

	return &logentriesAdaptor{
		defaultToken: route.Address,
		host:         dockerInfo.Name,
		instanceID:   instanceID,
		route:        route,
	}, nil
}

type LogentriesMessage struct {
	ID         string                 `json:"id,omitempty"`
	Image      string                 `json:"image,omitempty"`
	Created    string                 `json:"created,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Node       string                 `json:"node,omitempty"`
	InstanceID string                 `json:"instanceid,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Line       map[string]interface{} `json:"line,omitempty"`
}

func (adaptor *logentriesAdaptor) Stream(logstream chan *router.Message) {

	logger := NewLogger()

	for m := range logstream {

		token := m.Container.Config.Labels["logentries.token"]
		if token == "" {
			token = adaptor.defaultToken
		}

		var logline map[string]interface{}
		if err := json.Unmarshal([]byte(m.Data), &logline); err != nil {
			logline = make(map[string]interface{})
			logline["message"] = m.Data
		}

		jsonPayload, err := json.Marshal(LogentriesMessage{
			ID:         m.Container.Config.Hostname,
			Image:      m.Container.Config.Image,
			Created:    m.Container.Created.String(),
			Status:     m.Container.State.Status,
			Node:       adaptor.host,
			InstanceID: adaptor.instanceID,
			Labels:     m.Container.Config.Labels,
			Line:       logline,
		})
		if err != nil {
			log.Panic(err)
		}
		msg := []byte(token + " " + string(jsonPayload) + "\n")
		if err := logger.Write(&msg); err != nil {
			log.Println("could not write log message:", err)
		}
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
		nextBackoff = defaultBackoff
		tries := 0
	SendMessage:
		for {
			if conn == nil {
				conn, err = tls.Dial("tcp", "data.logentries.com:443", &tls.Config{})
				if err != nil {
					log.Println("open failed, backing off", nextBackoff, err)
					time.Sleep(nextBackoff)
					nextBackoff *= 2
					if nextBackoff > maxBackoff {
						nextBackoff = maxBackoff
					}
					continue
				}
			}
			totalWritten := 0
			for totalWritten < len(*msg) {
				written, err := conn.Write((*msg)[totalWritten:])
				if err != nil {
					tries += 1
					conn = nil
					log.Println("write failed, backing off", nextBackoff, err)
					time.Sleep(nextBackoff)
					nextBackoff *= 2
					if nextBackoff > maxBackoff {
						nextBackoff = maxBackoff
					}
					if tries >= 100 {
						log.Println("unalbe to send message after 100 tries, discarding")
						break SendMessage
					}
					continue SendMessage
				}
				nextBackoff = defaultBackoff
				totalWritten += written
			}
			// success
			break SendMessage
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
