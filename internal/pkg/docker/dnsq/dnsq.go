// Package dnsq creates temporary NSQ server containers.
package dnsq

import (
	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/docker"
	"github.com/nsqio/go-nsq"
)

// New creates and starts a new NSQ server.
func New() (addr string, cleanup func(), retErr error) {
	const port = "4150/tcp"
	container := &docker.Container{
		Repository: "nsqio/nsq",
		Tag:        "latest",
		Cmd:        []string{"/nsqd"},
		Ports:      []string{port},
		Ready: func(container *docker.Container) error {
			addr = container.GetHostPort(port)
			config := nsq.NewConfig()
			config.ClientID = uuid.Must(uuid.NewV4()).String()

			conn := nsq.NewConn(addr, config, (*nopDelegate)(nil))
			conn.SetLogger(nil, nsq.LogLevelInfo, "")
			defer conn.Close()

			// Connect sends IDENTIFY, so works as a ping.
			_, err := conn.Connect()
			return err
		},
		ExpirySecs: 300,
	}

	if err := container.Start(); err != nil {
		return "", nil, err
	}

	return addr, container.Cleanup, nil
}

type nopDelegate struct{}

func (*nopDelegate) OnResponse(c *nsq.Conn, data []byte)           {}
func (*nopDelegate) OnError(c *nsq.Conn, data []byte)              {}
func (*nopDelegate) OnMessage(c *nsq.Conn, m *nsq.Message)         {}
func (*nopDelegate) OnMessageFinished(c *nsq.Conn, m *nsq.Message) {}
func (*nopDelegate) OnMessageRequeued(c *nsq.Conn, m *nsq.Message) {}
func (*nopDelegate) OnBackoff(c *nsq.Conn)                         {}
func (*nopDelegate) OnContinue(c *nsq.Conn)                        {}
func (*nopDelegate) OnResume(c *nsq.Conn)                          {}
func (*nopDelegate) OnIOError(c *nsq.Conn, err error)              {}
func (*nopDelegate) OnHeartbeat(c *nsq.Conn)                       {}
func (*nopDelegate) OnClose(c *nsq.Conn)                           {}
