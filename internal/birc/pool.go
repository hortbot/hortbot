package birc

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hortbot/hortbot/internal/birc/breq"
	"github.com/hortbot/hortbot/internal/pkg/correlation"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/errgroupx"
	"github.com/hortbot/hortbot/internal/pkg/ircx"
	"github.com/jakebailey/irc"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	// DefaultMaxChannelsPerSubConn is the default maximum number of channels a
	// single subconn is allowed to be joined.
	DefaultMaxChannelsPerSubConn = 10

	// DefaultJoinRate is the default number of channels that can be joined by
	// pool per second.
	DefaultJoinRate = 1.0

	// DefaultPruneInterval is the default interval between subconn prunes.
	DefaultPruneInterval = time.Minute
)

// ErrPoolStopped is returned when the pool is stopped.
var ErrPoolStopped = errors.New("birc: pool stopped")

// Pool is a collection of managed IRC connections.
type Pool struct {
	config    *PoolConfig
	subConfig *Config

	ready chan struct{}

	recvChan chan *irc.Message
	sendChan chan breq.Send

	g *errgroupx.Group

	stopOnce sync.Once
	stopChan chan struct{}

	connID     uint64
	connsMu    sync.RWMutex
	conns      map[*Connection]struct{}
	chanToConn map[string]*Connection

	joinRate       time.Duration
	joinPartChan   chan breq.JoinPart
	syncJoinedChan chan breq.SyncJoined
	pruneChan      chan struct{}
	joinedMu       sync.RWMutex
	joined         map[string]bool
}

// NewPool creates a new Pool.
func NewPool(config PoolConfig) *Pool {
	config.Setup()

	subConfig := config.Config
	subConfig.InitialChannels = nil
	subConfig.RecvBuffer = 0

	p := &Pool{
		config:    &config,
		subConfig: &subConfig,

		ready: make(chan struct{}),

		stopChan: make(chan struct{}),
		recvChan: make(chan *irc.Message, config.Config.RecvBuffer),
		sendChan: make(chan breq.Send),

		conns:      make(map[*Connection]struct{}),
		chanToConn: make(map[string]*Connection),

		joinPartChan:   make(chan breq.JoinPart),
		syncJoinedChan: make(chan breq.SyncJoined),
		pruneChan:      make(chan struct{}, 1),
		joined:         make(map[string]bool),
	}

	if config.JoinRate > 0 {
		p.joinRate = time.Duration(float64(time.Second) / config.JoinRate)
	}

	return p
}

// Run runs the pool. It blocks until the pool is stopped, or the context
// cancelled. It always returns a non-nil error.
func (p *Pool) Run(ctx context.Context) error {
	defer close(p.recvChan)
	defer p.Stop()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	p.g = errgroupx.FromContext(ctx)

	p.g.Go(p.connManager)

	p.g.Go(func(ctx context.Context) error {
		select {
		case <-p.stopChan:
			return ErrPoolStopped
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	p.g.Go(func(ctx context.Context) error {
		return p.Join(ctx, p.config.InitialChannels...)
	})

	close(p.ready)

	return p.g.Wait()
}

// Stop stops the pool.
func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopChan)
	})
}

func (p *Pool) WaitUntilReady(ctx context.Context) error {
	select {
	case <-p.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Incoming returns a channel which is sent incoming messages. When the pool
// is stopped, this channel will be closed. Note that the returned channel is
// shared between all of the pool's subconnections and all other callers of
// Incoming, so it is imperative that this channel not be blocked needlessly.
func (p *Pool) Incoming() <-chan *irc.Message {
	return p.recvChan
}

// Join instructs the pool to join a channel.
func (p *Pool) Join(ctx context.Context, channels ...string) error {
	ctx = correlation.With(ctx)
	return p.doJoinPart(ctx, true, false, channels...)
}

// Part instructs the pool to part with a channel.
func (p *Pool) Part(ctx context.Context, channels ...string) error {
	ctx = correlation.With(ctx)
	return p.doJoinPart(ctx, false, false, channels...)
}

// Joined returns a sorted list of the joined channels.
func (p *Pool) Joined() []string {
	p.joinedMu.RLock()
	defer p.joinedMu.RUnlock()

	joined := make([]string, 0, len(p.joined))

	for ch := range p.joined {
		joined = append(joined, ch)
	}

	sort.Strings(joined)

	return joined
}

// IsJoined returns true if the specified channel has been joined.
func (p *Pool) IsJoined(channel string) bool {
	channel = ircx.NormalizeChannel(channel)

	if channel == "" {
		return false
	}

	p.joinedMu.RLock()
	defer p.joinedMu.RUnlock()
	return p.joined[channel]
}

// NumJoined returns the number of joined channels.
func (p *Pool) NumJoined() int {
	p.joinedMu.RLock()
	defer p.joinedMu.RUnlock()
	return len(p.joined)
}

// SyncJoined synchronizes the pool's joined channels to match the provided
// list.
func (p *Pool) SyncJoined(ctx context.Context, channels ...string) error {
	ctx = correlation.With(ctx)
	return breq.NewSyncJoined(channels).Do(ctx, p.syncJoinedChan, p.stopChan, ErrPoolStopped)
}

func (p *Pool) doJoinPart(ctx context.Context, join, force bool, channels ...string) error {
	if len(channels) == 0 {
		return nil
	}

	g := errgroupx.FromContext(ctx)

	for _, channel := range channels {
		channel := channel
		g.Go(func(ctx context.Context) error {
			return breq.NewJoinPart(channel, join, force).Do(ctx, p.joinPartChan, p.stopChan, ErrPoolStopped)
		})
	}

	return g.Wait()
}

func (p *Pool) connManager(ctx context.Context) error {
	var prune <-chan time.Time
	if p.config.PruneInterval > 0 {
		ticker := time.NewTicker(p.config.PruneInterval)
		defer ticker.Stop()
		prune = ticker.C
	}

	// Spawn at least one connection.
	if _, err := p.joinableConn(ctx); err != nil {
		return err
	}

	for {
		select {
		case req := <-p.joinPartChan:
			if err := p.handleJoinPart(ctx, req); err != nil {
				return err
			}

		case req := <-p.syncJoinedChan:
			if err := p.handleSyncJoined(ctx, req); err != nil {
				return err
			}

		case <-prune:
			p.prune(ctx)
			select {
			// Throw away any unneeded manual prune requests.
			case <-p.pruneChan:
			default:
			}

		case <-p.pruneChan:
			p.prune(ctx)
			select {
			// Throw away any unneeded automatic prune requests.
			case <-prune:
			default:
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// Only return an error if the entire pool should stop.
func (p *Pool) handleJoinPart(ctx context.Context, req breq.JoinPart) error {
	ctx = correlation.WithID(ctx, req.XID)
	err := p.joinPart(ctx, req.Channel, req.Join, req.Force)
	req.Finish(err)
	return p.joinSleep(ctx)
}

// Only return an error if the entire pool should stop.
func (p *Pool) handleSyncJoined(ctx context.Context, req breq.SyncJoined) error {
	ctx = correlation.WithID(ctx, req.XID)
	// TODO: Priorities for join orders.
	toPart, toJoin := p.joinPartChanges(req.Channels)

	// Part first so the existing connections are freed.
	for _, ch := range toPart {
		if err := p.joinPart(ctx, ch, false, true); err != nil {
			req.Finish(err)

			if cErr := ctx.Err(); cErr != nil {
				return cErr
			}

			return nil
		}
	}

	for _, ch := range toJoin {
		err := p.joinPart(ctx, ch, true, true)
		if err != nil {
			req.Finish(err)
		}

		if err := p.joinSleep(ctx); err != nil {
			return err
		}

		// req is already finished, return.
		if err != nil {
			return nil
		}
	}

	req.Finish(nil)
	return nil
}

func (p *Pool) joinSleep(ctx context.Context) error {
	if p.joinRate > 0 {
		select {
		case <-time.After(p.joinRate):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (p *Pool) joinPartChanges(want []string) ([]string, []string) {
	wantMap := make(map[string]bool, len(want))

	var toPart []string
	var toJoin []string

	toPartSeen := make(map[string]bool)
	toJoinSeen := make(map[string]bool)

	p.joinedMu.RLock()
	defer p.joinedMu.RUnlock()

	for _, ch := range want {
		wantMap[ch] = true

		if !p.joined[ch] && !toJoinSeen[ch] {
			toJoinSeen[ch] = true
			toJoin = append(toJoin, ch)
		}
	}

	for ch := range p.joined {
		if !wantMap[ch] && !toPartSeen[ch] {
			toPartSeen[ch] = true
			toPart = append(toPart, ch)
		}
	}

	sort.Strings(toPart)
	sort.Strings(toJoin)

	return toPart, toJoin
}

// joinPart joins or parts a channel if necessary, sleeping after joins.
func (p *Pool) joinPart(ctx context.Context, channel string, join bool, force bool) error {
	noChange := false

	if !force {
		p.joinedMu.RLock()
		noChange = p.joined[channel] == join
		p.joinedMu.RUnlock()
	}

	if noChange {
		return nil
	}

	if join {
		conn, err := p.joinableConn(ctx)
		if err != nil {
			return err
		}

		return p.join(ctx, conn, channel)
	}

	return p.part(ctx, channel)
}

func (p *Pool) join(ctx context.Context, conn *Connection, channel string) error {
	if err := conn.Join(ctx, channel); err != nil {
		return err
	}

	p.joinedMu.Lock()
	defer p.joinedMu.Unlock()
	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	p.joined[channel] = true
	p.chanToConn[channel] = conn

	return nil
}

func (p *Pool) part(ctx context.Context, channel string) error {
	p.connsMu.Lock()
	defer p.connsMu.Unlock()
	conn := p.chanToConn[channel]

	if conn == nil {
		ctxlog.Warn(ctx, "couldn't find conn for channel", zap.String("channel", channel))
		return nil
	}

	if err := conn.Part(ctx, channel); err != nil {
		return err
	}

	delete(p.chanToConn, channel)

	p.joinedMu.Lock()
	defer p.joinedMu.Unlock()
	delete(p.joined, channel)

	return nil
}

func (p *Pool) joinableConn(ctx context.Context) (*Connection, error) {
	conn := p.findJoinable()
	if conn != nil {
		return conn, nil
	}

	select {
	case conn := <-p.runSubConn():
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *Pool) findJoinable() *Connection {
	p.connsMu.RLock()
	defer p.connsMu.RUnlock()

	for conn := range p.conns {
		if conn.NumJoined() < p.config.MaxChannelsPerSubConn {
			return conn
		}
	}

	return nil
}

// Prune triggers a subconn prune.
func (p *Pool) Prune() {
	select {
	case p.pruneChan <- struct{}{}:
	default:
	}
}

func (p *Pool) prune(ctx context.Context) {
	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	toPrune := make([]*Connection, 0, 1)
	pruneAll := false

	for conn := range p.conns {
		if conn.NumJoined() == 0 {
			toPrune = append(toPrune, conn)
		} else {
			// Found one connection with channels, so prune any unused
			// connections that don't have any.
			pruneAll = true
		}
	}

	if len(toPrune) == 0 {
		return
	}

	if !pruneAll {
		toPrune = toPrune[1:]
	}

	if len(toPrune) == 0 {
		return
	}

	ctxlog.Debug(ctx, "pruning subconns", zap.Int("count", len(toPrune)))

	for _, conn := range toPrune {
		if err := conn.Close(); err != nil {
			ctxlog.Error(ctx, "error pruning subconn", zap.Error(err))
		}
		delete(p.conns, conn)
	}
}

func (p *Pool) NumConns() int {
	p.connsMu.Lock()
	defer p.connsMu.Unlock()
	return len(p.conns)
}

func (p *Pool) runSubConn() <-chan *Connection {
	newConn := make(chan *Connection)

	// This function should only return a non-nil error if the entire pool
	// needs to shut down. If a connection is closing, then it will queue
	// the channels to be joined and return nil.
	p.g.Go(func(ctx context.Context) (err error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		id := p.nextConnID()

		ctx = ctxlog.With(ctx, zap.Uint64("subconnID", id))

		ctxlog.Info(ctx, "spawning subconn")

		config := *p.subConfig
		conn := newConnection(&config)
		defer func() {
			ctxlog.Info(ctx, "closed subconn", zap.Error(conn.Close()))
		}()

		conn.sendFrom(p.sendChan)

		// No need to track this goroutine, when the pool or the connection
		// close, either the context will be cancelled, or the incoming channel
		// will be closed.
		go func() {
			incoming := conn.Incoming()

			for {
				var m *irc.Message
				var ok bool

				select {
				case m, ok = <-incoming:
					if !ok {
						return
					}
				case <-ctx.Done():
					return
				}

				select {
				case p.recvChan <- m:
					// Do nothing.
				case <-ctx.Done():
					return
				}
			}
		}()

		p.connsMu.Lock()
		p.conns[conn] = struct{}{}
		connsLen := len(p.conns)
		p.connsMu.Unlock()

		metricSubconns.WithLabelValues(p.config.UserConfig.Nick).Set(float64(connsLen))

		go func() {
			if err := conn.WaitUntilReady(ctx); err != nil {
				ctxlog.Warn(ctx, "waiting for connection to become ready", zap.Error(err))
				return
			}

			select {
			case newConn <- conn:
			case <-ctx.Done():
			}
		}()

		err = conn.Run(ctx)

		joined := conn.Joined()

		p.connsMu.Lock()
		p.joinedMu.Lock()
		delete(p.conns, conn)
		for _, channel := range joined {
			delete(p.chanToConn, channel)
			delete(p.joined, channel)
		}
		connsLen = len(p.conns)
		p.joinedMu.Unlock()
		p.connsMu.Unlock()

		metricSubconns.WithLabelValues(p.config.UserConfig.Nick).Set(float64(connsLen))

		// Context expired, keep returning.
		switch err {
		case context.Canceled, context.DeadlineExceeded:
			return err
		}

		// Ask the pool to join the lost channels, which will redistribute to
		// other open connections or spawn new ones. This is done on a
		// best-effort basis; the error below should only be returned if the
		// context was cancelled (since this context is the pool's context).
		err = p.doJoinPart(ctx, true, true, joined...)

		// Context expired, keep returning.
		switch err {
		case context.Canceled, context.DeadlineExceeded:
			return err
		}

		return nil
	})

	return newConn
}

func (p *Pool) nextConnID() uint64 {
	return atomic.AddUint64(&p.connID, 1)
}

func (p *Pool) send(ctx context.Context, m *irc.Message) error {
	return breq.NewSend(m).Do(ctx, p.sendChan, p.stopChan, ErrPoolStopped)
}

// SendMessage sends a PRIVMSG through the pool to a subconn.
//
// Note: this function does no rate limiting. Apply any rate limits before
// calling this function.
func (p *Pool) SendMessage(ctx context.Context, target, message string) error {
	ctx = correlation.With(ctx)
	return p.send(ctx, ircx.PrivMsg(target, message))
}

func (p *Pool) Quit(ctx context.Context) error {
	return p.send(ctx, &irc.Message{Command: "QUIT"})
}
