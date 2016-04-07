package stompy

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maleck13/stompy/Godeps/_workspace/src/github.com/nu7hatch/gouuid"
)

const _ErrorDisconnect string = "disconnected unexpectedly"
const _ServerError string = "server error "

var (
	ErrDisconnected                 error = errors.New("client disconnected unexpectedly")
	ErrPostConnectDisconnectHandler error = errors.New("a disconnect handler should be registered before calling connect")
	DefaultDisconnectHandler              = func(err error) {
		//hmmm what to do
		log.Println("defualt disconnect handler: ", err)
	}
)

const (
	STOMP_1_1 string = "1.1"
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

type ConnectionError string
type ServerError string
type BadFrameError string
type SubscriptionHandler func(Frame)

func (ce ConnectionError) Error() string {
	return _ErrorDisconnect + " : " + string(ce)
}

func (ce ServerError) Error() string {
	return _ServerError + " : " + string(ce)
}

func (be BadFrameError) Error() string {
	return "bad frame recieved from server : " + string(be)
}

//the disconnect handler is called on disconnect error from the network. It should handle trying to reconnect
//and set up the subscribers again
type DisconnectHandler func(error)

type StompConnector interface {
	Connect() error
	Disconnect() error
	RegisterDisconnectHandler(DisconnectHandler) error
}

type StompSubscriber interface {
	Subscribe(string, SubscriptionHandler, HEADERS) error
}

type StompPublisher interface {
	Send([]byte, string, string, HEADERS) error
}

//a stomp client is all of these things
type StompClient interface {
	StompConnector
	StompSubscriber
	StompPublisher
}

type subscriptions struct {
	sync.Mutex
	subs map[string]SubscriptionHandler
}

type Client struct {
	opts             ClientOpts
	connectionErr    chan error //we send an error on this channel if there is a connection error
	shutdown         chan bool  // tell any loops to etc for example the readLoop
	writeChannel     chan Frame // send things to the frame write to write to the server
	readChannel      chan Frame
	ReconnectHandler DisconnectHandler
	conn             net.Conn
	writer           *bufio.Writer
	reader           *bufio.Reader
	connectionLock   sync.Mutex // protects the client state when connecting
	_connecting      bool
	subLock          sync.Mutex
	subscriptions    *subscriptions
}

func NewClient(opts ClientOpts) StompClient {
	errChan := make(chan error)
	writeChan := make(chan Frame)
	readChan := make(chan Frame)
	shutdown := make(chan bool, 1)
	subMap := make(map[string]SubscriptionHandler)
	subs := &subscriptions{subs: subMap}
	return &Client{opts: opts, connectionErr: errChan, writeChannel: writeChan, readChannel: readChan, shutdown: shutdown, subscriptions: subs}
}

//StompConnector.Connect creates a tcp connection. sends any error through the errChan
func (client *Client) Connect() error {
	client.connectionLock.Lock()
	defer client.connectionLock.Unlock()
	client._connecting = true
	//set up default disconnect handler that just logs out the err
	if client.ReconnectHandler == nil {
		client.RegisterDisconnectHandler(DefaultDisconnectHandler)
	}
	conn, err := net.DialTimeout("tcp", client.opts.HostAndPort, client.opts.Timeout)
	if err != nil {
		client.connectionErr <- ConnectionError(err.Error())
		return ErrDisconnected
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
	if err := client.writeFrame(connectFrame); err != nil {
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
	fmt.Println("disconnect called")
	select {
	case client.shutdown <- true:
	default:

	}
	close(client.connectionErr)
	close(client.readChannel)
	close(client.writeChannel)
	close(client.shutdown)

	if nil != client.conn {
		return client.conn.Close()
	}
	return nil
}

//StompConnector.RegisterDisconnectHandler register a handler func that is sent any disconnect errors
func (client *Client) RegisterDisconnectHandler(handler DisconnectHandler) error {
	client.ReconnectHandler = handler
	if nil != client.conn {
		return ErrPostConnectDisconnectHandler
	}
	go func(errChan chan error) {
		//todo could end up with multiple handlers
		//todo prob dont want to fire this multiple times between disconnects. Likely needs more sophistication
		for err := range errChan {
			if _, ok := err.(ConnectionError); ok {
				client.ReconnectHandler(err)
			}
		}
	}(client.connectionErr)
	return nil
}

//StompPublisher.Send send a message to the server
func (client *Client) Send(body []byte, destination, contentType string, addedHeaders HEADERS) error {
	headers := sendHeaders(destination, contentType, addedHeaders)
	frame := NewFrame(_COMMAND_SEND, headers, body, client.connectionErr)
	//client.writeChannel <- frame
	//todo should it be async if so how to handle error. Should we stop any sending before connection is ready?
	return client.writeFrame(frame)
}

//subscribe to messages sent to the destination.
//headers are id and ack
func (client *Client) Subscribe(destination string, handler SubscriptionHandler, headers HEADERS) error {
	//create an id
	id, err := uuid.NewV4()
	if nil != err {
		return err
	}
	client.subscriptions.Lock()
	defer client.subscriptions.Unlock()
	client.subscriptions.subs[id.String()] = handler
	subHeaders := subscribeHeaders(id.String(), destination)
	frame := Frame{_COMMAND_SUBSCRIBE, subHeaders, _NULLBUFF, client.connectionErr}
	if err := client.writeFrame(frame); err != nil {
		return err
	}
	return nil
}

func (client *Client) writeFrame(frame Frame) error {

	frame.Headers["content-length"] = strconv.Itoa(len(frame.Body))
	if _, err := client.writer.Write(frame.Command); err != nil {
		//treating failure to write to the socket as a network error
		return ConnectionError(err.Error())
	}
	fmt.Println("writing frame ", frame)
	for k, v := range frame.Headers {
		val := k + ":" + v + "\n"
		if _, err := client.writer.WriteString(val); err != nil {
			return err
		}
	}
	if err := client.writer.WriteByte('\n'); err != nil {
		return err
	}
	if len(frame.Body) > 0 {
		if _, err := client.writer.Write(frame.Body); err != nil {
			return err
		}
	}
	if err := client.writer.Flush(); err != nil {
		return err
	}
	//stomp protocol want a null byte at the end of the frame
	if err := client.writer.WriteByte('\x00'); err != nil {
		return err
	}
	if err := client.writer.Flush(); err != nil {
		return err
	}
	return nil
}

//reads a single frame of the wire
func (client *Client) readFrame() (Frame, error) {
	f := Frame{}
	b, err := client.reader.ReadByte()
	fmt.Println("read byte ", b, err)
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
		f.Headers[parsed[0]] = parsed[1]
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
			fmt.Println("reading next frame")
			frame, err := client.readFrame()
			fmt.Println(frame, err)
			if err != nil {
				client.connectionErr <- ConnectionError("failed when reading frame " + err.Error())
			}
			fmt.Println("recieved frame in readLoop ", frame)
		}

	}

}

func (client *Client) writeLoop() {

	for {
		select {
		case f := <-client.writeChannel:
			client.writeFrame(f)
		}
	}

}
