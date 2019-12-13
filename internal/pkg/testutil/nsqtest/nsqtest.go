package nsqtest

import (
	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/docker"
	"github.com/nsqio/go-nsq"
)

func New() (addr string, cleanup func(), retErr error) {
	container := &docker.Container{
		Repository: "nsqio/nsq",
		Tag:        "latest",
		Cmd:        []string{"/nsqd"},
		Ready: func(container *docker.Container) error {
			addr = container.GetHostPort("4150/tcp")
			config := nsq.NewConfig()
			config.ClientID = uuid.Must(uuid.NewV4()).String()

			conn := nsq.NewConn(addr, config, &nopDelegate{})
			if _, err := conn.Connect(); err != nil {
				return err
			}
			return conn.Close()
		},
		ExpirySecs: 300,
	}

	if err := container.Start(); err != nil {
		return "", nil, err
	}

	return addr, container.Cleanup, nil
}

type nopDelegate struct{}

var _ nsq.ConnDelegate = (*nopDelegate)(nil)

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
