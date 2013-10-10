package moh

import (
	"code.google.com/p/go.net/websocket"
	"log"
)

// Publisher is the counterpart for Subscriber.
// It is a HTTP server accepting websocket connections.
type Publisher struct {
	MessagingServer

	// Registered filters, holds pointers to open connections.
	// All clients are registered to the "all" key by default for allowing broadcasting.
	// Modifier operations on this type is made by registrar() function.
	filters Filters

	// subscribe, disconnect events from connections
	events chan publisherEvent
}

// Subscription requests from connections to be sent to Publisher.subscribe channel
type publisherEvent struct {
	conn      *connection
	eventType int // values are defined as constants on global scope
	key       string
}

// This is the magic subscription key for broadcast events.
// Hoping that it is unique enough to not collide with another key.
const all = "4658f005d49885355f4e771ed9dace10cca9563e"

// Values for publisherEvent.eventType filed
const (
	subscribe = iota
	disconnect
	// unsubscribe event is not implemented yet
)

// NewPublisher creates a new Publisher and returns a pointer to it.
// The publisher will listen on addr and accept websocket connections from Subscribers.
func NewPublisher(addr string) (*Publisher, error) {
	s, err := NewMessagingServer(addr)
	if err != nil {
		return nil, err
	}

	p := &Publisher{
		MessagingServer: *s,
		filters:         make(Filters),
		events:          make(chan publisherEvent),
	}

	p.Mux.Handle("/", p.makeWsHandler())

	go s.Serve() // Starts HTTP server
	go p.registrar()

	return p, nil
}

// Publish sends a message to registered Subscribers with the key.
func (p *Publisher) Publish(key string, message []byte) {
	log.Println("Sending message to send channel")
	for c := range p.filters[key] {
		select {
		case c.send <- message:
			log.Println("Message sent to send channel")
		default:
			// Buffer is full, writer() is not fast enough to send all published messages .
			// Drop the websocket client and let it synchronize by re-connecting.
			log.Println("Websocket buffer is full. Dropping socket")
			go c.ws.Close()
		}
	}
}

// Broadcast sends a message to all of the connected Subscribers.
func (p *Publisher) Broadcast(message []byte) {
	p.Publish(all, message)
}

func (p *Publisher) makeWsHandler() websocket.Handler {
	return func(ws *websocket.Conn) {
		c := &connection{
			ws:   ws,
			send: make(chan []byte, 256),
			keys: make([]string, 0),
		}
		p.events <- publisherEvent{conn: c, eventType: subscribe, key: all}
		defer func() { p.events <- publisherEvent{conn: c, eventType: disconnect} }()
		go c.writer()
		c.reader(p.events)
	}
}

// registrar receives publiserEvents from the channel and updates filters.
// Adds or removes the connections from filters as if necessary.
// Synchronizes the modifier operations on Publisher.filters field.
func (p *Publisher) registrar() {
	for event := range p.events {
		switch event.eventType {
		case subscribe:
			p.filters.Add(event.key, event.conn)
		case disconnect:
			close(event.conn.send)
			p.filters.Remove(event.conn)
		}
	}
}

// connection represents a connected Subscriber in Publisher.
type connection struct {
	ws *websocket.Conn

	// Buffered channel of outbount messages
	send chan []byte

	// Subscription keys
	keys []string
}

// reader reads the subscription requests from websocket and saves it in a map for accessing later.
func (c *connection) reader(ch chan publisherEvent) {
	for {
		var key string
		err := websocket.Message.Receive(c.ws, &key)
		if err != nil {
			log.Println("reader: Cannot receive message from websocket")
			break
		}
		log.Println("reader: Received a message from websocket")
		ch <- publisherEvent{conn: c, eventType: subscribe, key: key}
	}
	c.ws.Close()
}

// writer writes the messages to the websocket from the send channel.
func (c *connection) writer() {
	for message := range c.send {
		err := websocket.Message.Send(c.ws, message)
		if err != nil {
			log.Println("writer: Cannot send message to websocket")
			break
		}
	}
	c.ws.Close()
}
