package stompy

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/maleck13/stompy/Godeps/_workspace/src/github.com/nu7hatch/gouuid"
)

const (
	STOMP_1_1 string = "1.1"
	STOMP_1_2 string = "1.2"
)

var (
	Supported                = []string{STOMP_1_1, STOMP_1_2}
	DefaultDisconnectHandler = func(err error) {
		//hmmm what to do
		log.Println("defualt disconnect handler: ", err)
	}
)

//wraps up various connection and auth params
type ClientOpts struct {
	Vhost       string
	HostAndPort string
	Timeout     time.Duration
	User        string
	PassCode    string
	Version     string
}

//the subscription handler type defines the function signature that should be passed when subscribing to queues
type SubscriptionHandler func(Frame)

//the disconnect handler is called on disconnect error from the network. It should handle trying to reconnect
//and set up the subscribers again
type DisconnectHandler func(error)

//responsible for defining the how the connection to the server should be handled
type StompConnector interface {
	Connect() error
	Disconnect() error
	RegisterDisconnectHandler(handler DisconnectHandler)
}

//responsible for defining how a subscription should be handled
type StompSubscriber interface {
	//accepts a destination /test/test for example a handler function for handling messages from that subscription and any headers you want to override / set
	Subscribe(destination string, handler SubscriptionHandler, headers HEADERS) error
}

//responsible for defining how a publish should happen
type StompPublisher interface {
	//accepts a body, destination, content-type and any headers you wish to override or set
	Publish([]byte, string, string, HEADERS) error
}

//A stomp client is a combination of all of these things
type StompClient interface {
	StompConnector
	StompSubscriber
	StompPublisher
}

//lockable struct for mapping subscription ids to their handlers
type subscriptions struct {
	sync.Mutex
	subs map[string]SubscriptionHandler
}

type Client struct {
	opts              ClientOpts
	connectionErr     chan error        //we send an error on this channel if there is a connection error. the DisconnectHandler is called if this channel receives an error
	shutdown          chan bool         // tell any loops to exit as we are disconnecting. For example the readLoop
	DisconnectHandler DisconnectHandler // a func that should do something in the case of a network disconnection
	conn              net.Conn
	writer            *bufio.Writer //used to write to the network socket
	reader            *bufio.Reader //used to read from the network socket
	connectionLock    sync.Mutex    // protects the client state while connecting
	_connecting       bool          //flag for to let the client handle actions performed before connection is established
	subscriptions     *subscriptions
}

func NewClient(opts ClientOpts) StompClient {
	errChan := make(chan error)
	shutdown := make(chan bool, 1)
	subMap := make(map[string]SubscriptionHandler)
	subs := &subscriptions{subs: subMap}
	return &Client{opts: opts, connectionErr: errChan, shutdown: shutdown, subscriptions: subs}
}

//StompConnector.Connect creates a tcp connection. sends any error through the errChan also returns the error
func (client *Client) Connect() error {
	//make sure other methods understand that we are currently connecting
	client.connectionLock.Lock()
	defer client.connectionLock.Unlock()
	client._connecting = true
	//set up default disconnect handler that just logs out the err
	if client.DisconnectHandler == nil {
		client.RegisterDisconnectHandler(DefaultDisconnectHandler)
	}
	conn, err := net.DialTimeout("tcp", client.opts.HostAndPort, client.opts.Timeout)
	if err != nil {
		connErr := ConnectionError(err.Error())
		client.connectionErr <- connErr
		return connErr
	}

	client.conn = conn
	//set up a buffered writer and reader for our socket
	client.writer = bufio.NewWriter(conn)
	client.reader = bufio.NewReader(conn)

	headers, err := connectionHeaders(client.opts)
	if err != nil {
		return ConnectionError(err.Error())
	}
	connectFrame := NewFrame(_COMMAND_CONNECT, headers, _NULLBUFF, client.connectionErr)
	if err := writeFrame(client.writer,connectFrame); err != nil {
		client.sendConnectionError(err)
		return err
	}

	//read frame after writing out connection to check we are connected
	if _, err = client.readFrame(); err != nil {
		client.sendConnectionError(err)
		return err
	}
	//start background readloop
	go client.readLoop()
	return nil

}

func (client *Client) sendConnectionError(err error) {
	if _, is := err.(ConnectionError); is {
		select {
		case client.connectionErr <- err:
		default:
		}
	}
}

//StompConnector.Disconnect close our error channel then close the socket connection
func (client *Client) Disconnect() error {
	//signal read loop to shutdown
	select {
	case client.shutdown <- true:
	default:

	}
	close(client.connectionErr)
	close(client.shutdown)

	if nil != client.conn {
		return client.conn.Close()
	}
	return nil
}

//StompConnector.RegisterDisconnectHandler register a handler func that is sent any disconnect errors
func (client *Client) RegisterDisconnectHandler(handler DisconnectHandler) {
	client.DisconnectHandler = handler
	go func(errChan chan error) {
		//todo could end up with multiple handlers
		//todo prob dont want to fire this multiple times between disconnects. Likely needs more sophistication
		for err := range errChan {
			if _, ok := err.(ConnectionError); ok {
				client.DisconnectHandler(err)
			}
		}
	}(client.connectionErr)
}

//StompPublisher.Send publish a message to the server
func (client *Client) Publish(body []byte, destination, contentType string, addedHeaders HEADERS) error {
	headers := sendHeaders(destination, contentType, addedHeaders)
	frame := NewFrame(_COMMAND_SEND, headers, body, client.connectionErr)
	//todo should it be async if so how to handle error. Should we stop any sending before connection is ready?
	return writeFrame(client.writer,frame)
}

//subscribe to messages sent to the destination.
//headers are id and ack
func (client *Client) Subscribe(destination string, handler SubscriptionHandler, headers HEADERS) error {
	//create an id
	id, err := uuid.NewV4()
	if nil != err {
		return err
	}
	idStr := id.String()
	//ensure we don't end up with double registration
	client.subscriptions.Lock()
	defer client.subscriptions.Unlock()
	if _, ok := client.subscriptions.subs[idStr]; ok {
		return ClientError("collision with subscription id")
	}
	client.subscriptions.subs[idStr] = handler
	subHeaders := subscribeHeaders(idStr, destination)
	frame := Frame{_COMMAND_SUBSCRIBE, subHeaders, _NULLBUFF}
	if err := writeFrame(client.writer,frame); err != nil {
		return err
	}
	return nil
}

//reads a single frame of the wire
func (client *Client) readFrame() (Frame, error) {
	f := Frame{}
	line, err := client.reader.ReadBytes('\n')
	//count this as a connection error. will be sent via error channel to the reconnect handler
	if err != nil {
		return f, ConnectionError(err.Error())
	}
	f.Command = line
	//sort out our headers
	f.Headers = make(map[string]string)
	for {
		header, err := client.reader.ReadString('\n')
		if nil != err {
			return f, err
		}
		if header == "\n" {
			//reached end of headers break should we set some short deadlock break?
			break
		}
		parsed := strings.SplitN(header, ":", 2)
		if len(parsed) != 2 {
			return f, BadFrameError("failed to parse header correctly " + header)
		}
		//todo need to decode the headers
		f.Headers[strings.TrimSpace(parsed[0])] = strings.TrimSpace(parsed[1])
	}
	//ready body
	body, err := client.reader.ReadBytes('\n')
	if err != nil {
		return f, err
	}
	f.Body = body[0 : len(body)-1]

	if string(f.Command) == "ERROR\n" {
		if client._connecting {
			return f, ServerError("error returned from server during initial connection. " + f.Headers["message"])
		}
		return f, ServerError("error returned from server: " + f.Headers["message"])
	}
	return f, nil
}

func (client *Client) readLoop() {
	//read a frame, if it is a subscription send it be handled
	for {
		select {
		case <-client.shutdown:
			return
		default:
			frame, err := client.readFrame() //this will block until it reads
			if err != nil {
				client.connectionErr <- ConnectionError("failed when reading frame " + err.Error())
			}
			go func(f Frame) {
				//coming from the server it will be one of
				/*
					"CONNECTED"
					"MESSAGE"
					"RECEIPT"
					"ERROR"
				*/
				cmd := string(f.Command)
				fmt.Println(f.Headers)
				if sub, ok := f.Headers["subscription"]; ok {
					if _, hasSub := client.subscriptions.subs[sub]; hasSub {
						client.subscriptions.subs[sub](f)
					}
				}
				fmt.Println("command read ", cmd)
			}(frame)
		}

	}

}
