package amqp

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	driver "github.com/rabbitmq/amqp091-go"
	"go.bryk.io/pkg/errors"
	xlog "go.bryk.io/pkg/log"
)

// Return captures a flattened struct of fields returned by the server when a
// publish operation is unable to be delivered either due to the "mandatory"
// flag set and no route found, or "immediate" flag set and no free consumer.
type Return = driver.Return

const (
	// When reconnecting to the server after connection failure.
	reconnectDelay = 3 * time.Second

	// When resending messages the server didn't confirm.
	resendDelay = 3 * time.Second

	// Time to wait for a user to receive an ACK notification when
	// publishing messages to the broker.
	ackDelay = 10 * time.Millisecond
)

// Common errors.
var (
	errShutdown        = "session is shutting down"
	errNotConnected    = "not connected to a server"
	errAlreadyClosed   = "session is already closed"
	errUnconfirmedPush = "unconfirmed push"
)

// Session instances handle an underlying connection and channel with an
// AMQP server. Providing topology setup and automatic reconnection.
type session struct {
	topology        Topology                 // expected broker topology settings
	name            string                   // entity identifier
	addr            string                   // broker endpoint
	log             xlog.Logger              // internal logger
	conn            *driver.Connection       // broker connection
	channel         *driver.Channel          // broker communication channel
	tlsConf         *tls.Config              // TLS settings when using AMQPS
	reconnect       chan bool                // internal listener for reconnect attempts
	notifyConnClose chan *driver.Error       // listener for connection close events
	notifyChanClose chan *driver.Error       // listener for channel or connection exceptions
	notifyConfirm   chan driver.Confirmation // listener for reliable publishing confirmations
	notifyReturn    chan Return              // listener for undeliverable message events
	prefetchCount   int                      // prefetch by message count
	prefetchSize    int                      // prefetch by bytes flushed to the network
	status          chan bool                // listener for 'readiness' state updates
	rpcEnabled      bool                     // whether RPC style operations are supported
	rr              bool                     // readiness session state
	wg              *sync.WaitGroup          // background tasks counter
	mc              []chan<- bool            // in-flight message confirmation listeners
	mr              []chan<- Return          // in-flight message return listeners
	mu              sync.RWMutex
	ctx             context.Context
	halt            context.CancelFunc
}

// Open a new session instance.
func open(addr string, options ...Option) (*session, error) {
	// Base session instance
	ctx, halt := context.WithCancel(context.Background())
	s := &session{
		addr:          addr,
		reconnect:     make(chan bool, 5),
		status:        make(chan bool, 1),
		prefetchSize:  0,
		prefetchCount: 1,
		halt:          halt,
		ctx:           ctx,
		log:           xlog.Discard(),
		wg:            new(sync.WaitGroup),
		mc:            []chan<- bool{},
		mr:            []chan<- Return{},
	}
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	if s.name == "" {
		s.name = getName("session")
	}

	// Automatically start event processing in the background
	go s.eventLoop()
	s.reconnect <- true
	return s, nil
}

// Close will cleanly shut down the channel and connection.
func (s *session) close() error {
	// Already closed?
	if !s.isReady() {
		return errors.New(errAlreadyClosed)
	}

	// Stop event processing
	s.log.Debug("closing session")
	s.halt()
	<-s.ctx.Done()

	// Gracefully close channel
	if err := s.channel.Close(); err != nil {
		return err
	}

	// Gracefully close connection
	if err := s.conn.Close(); err != nil {
		return err
	}
	s.updateStatus(false)
	s.wg.Wait()
	s.clean()
	return nil
}

// Free resources no longer needed when a session is closed.
func (s *session) clean() {
	// Close all notifications handlers
	s.mu.Lock()
	for _, ack := range s.mc {
		close(ack)
	}
	for _, mr := range s.mr {
		close(mr)
	}
	close(s.status)
	s.mu.Unlock()
}

// Return the current readiness session state.
func (s *session) isReady() bool {
	s.mu.RLock()
	v := s.rr
	s.mu.RUnlock()
	return v
}

// Update the readiness state for the session instance.
func (s *session) updateStatus(value bool) {
	s.mu.Lock()
	s.rr = value
	s.mu.Unlock()

	// Notify readiness status
	s.wg.Add(1)
	go func(val bool) {
		defer s.wg.Done()
		select {
		case s.status <- val:
			return
		case <-s.ctx.Done():
			return
		case <-time.After(ackDelay):
			return
		}
	}(value)
}

// Prepare AMQP connection and state.
func (s *session) init() error {
	if s.conn == nil || s.conn.IsClosed() {
		// Open new connection
		conn, err := driver.DialTLS(s.addr, s.tlsConf)
		if err != nil {
			return err
		}

		// Set connection in the session instance
		s.setConnection(conn)
		s.log.Info("connected")
	}

	// Open a new channel with the server
	ch, err := s.conn.Channel()
	if err != nil {
		return err
	}

	// Setup channel
	if err = ch.Qos(s.prefetchCount, s.prefetchSize, false); err != nil {
		return err
	}
	if err = ch.Confirm(false); err != nil {
		return err
	}

	// Ensure broker topology
	if err = s.loadTopology(ch); err != nil {
		return err
	}

	// Set channel and mark session as ready
	s.setChannel(ch)
	s.updateStatus(true)
	s.log.Info("ready")
	return nil
}

// Set the active AMQP connection on the session instance.
func (s *session) setConnection(conn *driver.Connection) {
	// Update connection and related listeners
	s.mu.Lock()
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.conn = conn
	s.notifyConnClose = make(chan *driver.Error)
	s.conn.NotifyClose(s.notifyConnClose)
	s.mu.Unlock()
}

// Set the active AMQP channel on the session instance.
func (s *session) setChannel(channel *driver.Channel) {
	// Update channel and related listeners
	s.mu.Lock()
	s.channel = channel
	s.notifyChanClose = make(chan *driver.Error)
	s.notifyConfirm = make(chan driver.Confirmation, 10)
	s.notifyReturn = make(chan driver.Return, 10)
	s.channel.NotifyClose(s.notifyChanClose)
	s.channel.NotifyPublish(s.notifyConfirm)
	s.channel.NotifyReturn(s.notifyReturn)
	s.mu.Unlock()
}

// Ensure the broker topology matches the user expectations. Missing
// entities will be created.
func (s *session) loadTopology(ch *driver.Channel) error {
	for _, ex := range s.topology.Exchanges {
		if err := s.addExchange(ex, ch); err != nil {
			return err
		}
	}
	for _, q := range s.topology.Queues {
		if _, err := s.addQueue(q, ch); err != nil {
			return err
		}
	}
	for _, b := range s.topology.Bindings {
		if err := s.addBinding(b, ch); err != nil {
			return err
		}
	}
	return nil
}

// Register an exchange declaration with the provided channel.
func (s *session) addExchange(ex Exchange, ch *driver.Channel) error {
	return ch.ExchangeDeclare(
		ex.Name,
		ex.Kind,
		ex.Durable,
		ex.AutoDelete,
		ex.Internal,
		false,
		ex.Arguments)
}

// Register a queue declaration with the provided channel.
func (s *session) addQueue(q Queue, ch *driver.Channel) (string, error) {
	// Generate a random name for the queue if not provided.
	// The name is prefixed with the session's name for easy
	// filtering.
	if q.Name == "" {
		q.Name = getName(fmt.Sprintf("%s-gen", s.name))
	}
	_, err := ch.QueueDeclare(
		q.Name,
		q.Durable,
		q.AutoDelete,
		q.Exclusive,
		false,
		q.Arguments)
	return q.Name, err
}

// Register a binding declaration with the provided channel.
func (s *session) addBinding(b Binding, ch *driver.Channel) error {
	if len(b.RoutingKey) > 0 {
		for _, rk := range b.RoutingKey {
			err := ch.QueueBind(
				b.Queue,
				rk,
				b.Exchange,
				false,
				b.Arguments)
			if err != nil {
				return err
			}
		}
	} else {
		return ch.QueueBind(
			b.Queue,
			"",
			b.Exchange,
			false,
			b.Arguments)
	}
	return nil
}

// Register a one-off receiver for publishing a confirmation.
func (s *session) ack() <-chan bool {
	ch := make(chan bool)
	s.mu.Lock()
	s.mc = append(s.mc, ch)
	s.mu.Unlock()
	return ch
}

// Return a monitor to receive notifications for messages returned
// by the broker.
func (s *session) messageReturns() <-chan Return {
	monitor := make(chan Return)
	s.mu.Lock()
	s.mr = append(s.mr, monitor)
	s.mu.Unlock()
	return monitor
}

// Process message confirmations received as publish notifications.
func (s *session) handleConfirmation(msg driver.Confirmation) {
	// Ignore "empty" confirmations
	if msg.DeliveryTag == 0 {
		return
	}

	// No ack listener registered
	s.mu.Lock()
	if len(s.mc) == 0 {
		s.mu.Unlock()
		return
	}

	// Pop last ACK entry
	index := len(s.mc) - 1
	ack := s.mc[index]
	s.mc = s.mc[:index]
	s.mu.Unlock()

	// Deliver ACK result on the background
	s.wg.Add(1)
	go func(ctx context.Context, ack chan<- bool) {
		defer s.wg.Done()
		select {
		case ack <- msg.Ack:
			break
		case <-time.After(ackDelay):
			break
		case <-ctx.Done():
			break
		}
		close(ack)
	}(s.ctx, ack)
}

// Process messages returned from the server.
func (s *session) handleMessageReturns(msg Return) {
	s.mu.Lock()
	for _, m := range s.mr {
		// Deliver message return on the background
		s.wg.Add(1)
		go func(ctx context.Context, m chan<- Return) {
			defer s.wg.Done()
			select {
			case m <- msg:
				return
			case <-time.After(ackDelay):
				return
			case <-ctx.Done():
				return
			}
		}(s.ctx, m)
	}
	s.mu.Unlock()
}

// Handle all internal event processing for the session.
func (s *session) eventLoop() {
	for {
		select {
		// Terminate event processing.
		// Connection was manually closed, no automatic reconnection is required.
		case <-s.ctx.Done():
			s.log.Debug("stop listening for session events")
			return
		// Catch connection errors.
		case _, ok := <-s.notifyConnClose:
			if !ok {
				// Connection was manually closed, no automatic reconnection is required.
				continue
			}
			if s.isReady() {
				// Unexpected disconnect, start automatic reconnection.
				s.log.Warning("connection closed")
				s.reconnect <- true
			}
		// Catch channel error. Start automatic reconnection.
		case _, ok := <-s.notifyChanClose:
			if !ok {
				// Connection was manually closed, no automatic reconnection is required.
				continue
			}
			if s.isReady() {
				// Unexpected disconnect, start automatic reconnection.
				s.log.Warning("channel closed")
				s.reconnect <- true
			}
		// Message published confirmations.
		case mc, ok := <-s.notifyConfirm:
			if ok {
				s.handleConfirmation(mc)
			}
		// Message returned notifications.
		case mr, ok := <-s.notifyReturn:
			if ok {
				s.handleMessageReturns(mr)
			}
		// Handle reconnections.
		case <-s.reconnect:
			s.updateStatus(false)
			s.log.Debug("attempting to connect")
			if err := s.init(); err != nil {
				s.log.Warning("failed to connect")
				<-time.After(reconnectDelay)
				s.reconnect <- true
			}
		}
	}
}
