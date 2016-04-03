package stompy

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

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

type DisconnectHandler func(error)

type StompConnector interface {
	Connect() error
	Disconnect() error
	RegisterDisconnectHandler(DisconnectHandler) error
}

type StompSubscriber interface {
}

type StompPublisher interface {
	Send([]byte, string, string, HEADERS)error
}

//a stomp client is all of these things
type StompClient interface {
	StompConnector
	StompSubscriber
	StompPublisher
}

type Client struct {
	opts             ClientOpts
	connectionErr    chan error
	ReconnectHandler DisconnectHandler
	conn             net.Conn
	writer           *bufio.Writer
	reader           *bufio.Reader
	writeChannel     chan Frame
	readChannel      chan Frame
}

func NewClient(opts ClientOpts) StompClient {
	errChan := make(chan error)
	writeChan := make(chan Frame)
	readChan := make(chan Frame)
	return &Client{opts: opts, connectionErr: errChan,writeChannel:writeChan,readChannel:readChan}
}

//StompConnector.Connect creates a tcp connection. sends any error through the errChan
func (client *Client) Connect() error {
	//set up default disconnect handler that just logs out the err
	if client.ReconnectHandler == nil {
		client.RegisterDisconnectHandler(DefaultDisconnectHandler)
	}
	conn, err := net.DialTimeout("tcp", client.opts.HostAndPort, client.opts.Timeout)
	if err != nil {
		client.connectionErr <- ErrDisconnected
		return ErrDisconnected
	}

	client.conn = conn
	//set up a buffered writer and reader for our socket
	client.writer = bufio.NewWriter(conn)
	client.reader = bufio.NewReader(conn)

	headers, err := connectionHeaders(client.opts)
	if err != nil {
		return err
	}
	connectFrame := NewFrame(_COMMAND_CONNECT, headers, _NULLBUFF,client.connectionErr)
	client.writeFrame(connectFrame)
	b, e := client.reader.ReadBytes(0)
	if e != nil {
		fmt.Println("returned from connect ", string(b))
		return e
	}
	return nil

}

//StompConnector.Disconnect close our error channel then close the socket connection
func (client *Client) Disconnect() error {
	close(client.connectionErr)
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
			if err == ErrDisconnected {
				client.ReconnectHandler(err)
			}
		}
	}(client.connectionErr)
	return nil
}

//StompPublisher.Send send a message to the server
func (client *Client)Send(body []byte, destination, contentType string, addedHeaders HEADERS)error{
	headers:= sendHeaders(destination,contentType,addedHeaders)
	frame := NewFrame(_COMMAND_SEND,headers,body,client.connectionErr)
	client.writeChannel <-frame
	return nil
}


func (client *Client) writeFrame(frame Frame)error {
	frame.Headers["content-length"] = strconv.Itoa(len(frame.Body))
	if _, err := client.writer.Write(frame.Command); err != nil {
		return err
	}

	for k, v := range frame.Headers {
		val := k + ":" + v + "\n"
		if _, err := client.writer.WriteString(val); err != nil {
			return err
		}
	}
	if err := client.writer.WriteByte('\n'); err != nil{
		return err
	}
	if len(frame.Body) > 0 {
		if _,err := client.writer.Write(frame.Body); err != nil{
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

func (client *Client) readFrame()Frame{
	return Frame{}
}

func (client *Client) readLoop(){

}


func (client *Client)writeLoop(){

	for{
		select {
		case f := <- client.writeChannel:
			client.writeFrame(f)
		}
	}

}
