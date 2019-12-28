package dnsq

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

			conn := nsq.NewConn(addr, config, (*nopDelegate)(nil))
			if _, err := conn.Connect(); err != nil {
				return err
			}
			defer conn.Close()

			return conn.WriteCommand(nsq.Nop())
		},
		ExpirySecs: 300,
	}

	if err := container.Start(); err != nil {
		return "", nil, err
	}

	return addr, container.Cleanup, nil
}

type nopDelegate struct {
	nsq.ConnDelegate // Embed and manually implement functions that the above code actually uses.
}

func (*nopDelegate) OnClose(c *nsq.Conn) {}