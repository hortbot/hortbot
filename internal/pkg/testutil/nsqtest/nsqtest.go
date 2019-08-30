package nsqtest

import (
	"github.com/nsqio/go-nsq"
	"github.com/ory/dockertest"
)

func New() (addr string, cleanup func(), retErr error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return "", nil, err
	}

	opts := &dockertest.RunOptions{
		Repository: "nsqio/nsq",
		Tag:        "latest",
		Cmd:        []string{"/nsqd"},
	}

	resource, err := pool.RunWithOptions(opts)
	if err != nil {
		return "", nil, err
	}

	defer func() {
		if retErr != nil {
			pool.Purge(resource) //nolint:errcheck
		}
	}()

	// Ensure the container is cleaned up, even if the process exits.
	if err := resource.Expire(300); err != nil {
		return "", nil, err
	}

	addr = resource.GetHostPort("4150/tcp")

	err = pool.Retry(func() error {
		conn := nsq.NewConn(addr, nsq.NewConfig(), &nopDelegate{})
		if _, err := conn.Connect(); err != nil {
			return err
		}
		return conn.Close()
	})
	if err != nil {
		return "", nil, err
	}

	return addr, func() {
		pool.Purge(resource) //nolint:errcheck
	}, nil
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
