package breq

import (
	"context"
	"errors"
	"testing"

	"github.com/fortytw2/leaktest"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/jakebailey/irc"
	"gotest.tools/v3/assert"
)

var errTest = errors.New("test error")

func TestSend(t *testing.T) {
	m := &irc.Message{
		Command:  "TEST",
		Trailing: "This is a test.",
	}

	t.Run("New", func(t *testing.T) {
		req := NewSend(m)
		assert.Assert(t, req.errChan != nil)
		assert.Assert(t, cap(req.errChan) == 1)
		assert.Assert(t, len(req.errChan) == 0)
		assert.Equal(t, m, req.M)
	})

	t.Run("Finish", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
		}{
			{"nil error", nil},
			{"non-nil error", errTest},
		}

		for _, test := range tests {
			test := test

			t.Run(test.name, func(t *testing.T) {
				defer leaktest.Check(t)()
				done := make(chan struct{})

				ctx := context.Background()

				req := NewSend(m)
				reqChan := make(chan Send)

				go func() {
					defer close(done)
					got := <-reqChan
					assert.Equal(t, req, got)
					got.Finish(test.err)
				}()

				err := req.Do(ctx, reqChan, nil, nil)
				assert.Equal(t, test.err, err)
				<-done
			})
		}
	})

	cancelTests := []struct {
		name       string
		send, stop bool
	}{
		{"Cancel during send", false, false},
		{"Stop during send", false, true},
		{"Cancel during err receive", true, false},
		{"Stop during err receive", true, true},
	}

	for _, test := range cancelTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			done := make(chan struct{})

			ctx, cancel := context.WithCancel(context.Background())

			req := NewSend(m)
			reqChan := make(chan Send)
			stopChan := make(chan struct{})

			go func() {
				defer close(done)

				if test.send {
					got := <-reqChan
					assert.Equal(t, req, got)
				}

				if test.stop {
					close(stopChan)
				} else {
					cancel()
				}
			}()

			err := req.Do(ctx, reqChan, stopChan, errTest)

			if test.stop {
				assert.Equal(t, errTest, err)
			} else {
				assert.Equal(t, context.Canceled, err)
			}

			<-done
		})
	}
}

func BenchmarkSend(b *testing.B) {
	ctx := context.Background()
	reqChan := make(chan Send)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for req := range reqChan {
			req.Finish(nil)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := NewSend(nil)
		_ = req.Do(ctx, reqChan, nil, nil)
	}
	close(reqChan)

	<-done
}

func BenchmarkSendStop(b *testing.B) {
	ctx := context.Background()
	reqChan := make(chan Send)
	stopChan := make(chan struct{})
	done := make(chan struct{})
	close(stopChan)

	go func() {
		defer close(done)
		for req := range reqChan {
			req.Finish(nil)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := NewSend(nil)
		_ = req.Do(ctx, reqChan, stopChan, nil)
	}
	close(reqChan)

	<-done
}

// TODO: the rest of these tests.

func TestJoinPart(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		tests := []struct {
			channel     string
			join, force bool
		}{
			{"foo", false, false},
			{"#bar", false, true},
			{"", true, false},
			{"hello", true, true},
		}

		for _, test := range tests {
			req := NewJoinPart(test.channel, test.join, test.force)
			assert.Assert(t, req.errChan != nil)
			assert.Assert(t, cap(req.errChan) == 1)
			assert.Assert(t, len(req.errChan) == 0)
			assert.Equal(t, ircx.NormalizeChannel(test.channel), req.Channel)
			assert.Equal(t, test.join, req.Join)
			assert.Equal(t, test.force, req.Force)
		}
	})

	t.Run("Finish", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
		}{
			{"nil error", nil},
			{"non-nil error", errTest},
		}

		for _, test := range tests {
			test := test

			t.Run(test.name, func(t *testing.T) {
				defer leaktest.Check(t)()
				done := make(chan struct{})

				ctx := context.Background()

				req := NewJoinPart("#foobar", true, false)
				reqChan := make(chan JoinPart)

				go func() {
					defer close(done)
					got := <-reqChan
					assert.Equal(t, req, got)
					got.Finish(test.err)
				}()

				err := req.Do(ctx, reqChan, nil, nil)
				assert.Equal(t, test.err, err)
				<-done
			})
		}
	})

	cancelTests := []struct {
		name       string
		send, stop bool
	}{
		{"Cancel during send", false, false},
		{"Stop during send", false, true},
		{"Cancel during err receive", true, false},
		{"Stop during err receive", true, true},
	}

	for _, test := range cancelTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			done := make(chan struct{})

			ctx, cancel := context.WithCancel(context.Background())

			req := NewJoinPart("#foobar", true, false)
			reqChan := make(chan JoinPart)
			stopChan := make(chan struct{})

			go func() {
				defer close(done)

				if test.send {
					got := <-reqChan
					assert.Equal(t, req, got)
				}

				if test.stop {
					close(stopChan)
				} else {
					cancel()
				}
			}()

			err := req.Do(ctx, reqChan, stopChan, errTest)

			if test.stop {
				assert.Equal(t, errTest, err)
			} else {
				assert.Equal(t, context.Canceled, err)
			}

			<-done
		})
	}
}

func TestSyncJoined(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		tests := [][]string{
			nil,
			{},
			{"foo", "bar"},
			{"#foobar"},
		}

		for _, test := range tests {
			req := NewSyncJoined(test)
			assert.Assert(t, req.errChan != nil)
			assert.Assert(t, cap(req.errChan) == 1)
			assert.Assert(t, len(req.errChan) == 0)
			assert.DeepEqual(t, ircx.NormalizeChannels(test...), req.Channels)
		}
	})

	t.Run("Finish", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
		}{
			{"nil error", nil},
			{"non-nil error", errTest},
		}

		for _, test := range tests {
			test := test

			t.Run(test.name, func(t *testing.T) {
				defer leaktest.Check(t)()
				done := make(chan struct{})

				ctx := context.Background()

				req := NewSyncJoined([]string{"#foobar"})
				reqChan := make(chan SyncJoined)

				go func() {
					defer close(done)
					got := <-reqChan
					got.Finish(test.err)

					// Check members; DeepEqual causes a hang.
					assert.Equal(t, req.errChan, got.errChan)
					assert.DeepEqual(t, req.Channels, got.Channels)
				}()

				err := req.Do(ctx, reqChan, nil, nil)
				assert.Equal(t, test.err, err)
				<-done
			})
		}
	})

	cancelTests := []struct {
		name       string
		send, stop bool
	}{
		{"Cancel during send", false, false},
		{"Stop during send", false, true},
		{"Cancel during err receive", true, false},
		{"Stop during err receive", true, true},
	}

	for _, test := range cancelTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			defer leaktest.Check(t)()
			done := make(chan struct{})

			ctx, cancel := context.WithCancel(context.Background())

			req := NewSyncJoined([]string{"#foobar"})
			reqChan := make(chan SyncJoined)
			stopChan := make(chan struct{})

			go func() {
				defer close(done)

				if test.send {
					got := <-reqChan

					// Check members; DeepEqual causes a hang.
					assert.Equal(t, req.errChan, got.errChan)
					assert.DeepEqual(t, req.Channels, got.Channels)
				}

				if test.stop {
					close(stopChan)
				} else {
					cancel()
				}
			}()

			err := req.Do(ctx, reqChan, stopChan, errTest)

			if test.stop {
				assert.Equal(t, errTest, err)
			} else {
				assert.Equal(t, context.Canceled, err)
			}

			<-done
		})
	}
}
