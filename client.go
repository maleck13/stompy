package stompy

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
	"sync"
	"strings"
)

const _ErrorDisconect string = "disconnected unexpectedly"
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

func (ce ConnectionError) Error()string {
	return _ErrorDisconect + " : " + string(ce)
}

func (ce ServerError) Error()string {
	return _ServerError + " : " + string(ce)
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
	connectionLock	 sync.Mutex // protects the client state when connecting
	_connecting      bool
}

func NewClient(opts ClientOpts) StompClient {
	errChan := make(chan error)
	writeChan := make(chan Frame)
	readChan := make(chan Frame)
	return &Client{opts: opts, connectionErr: errChan,writeChannel:writeChan,readChannel:readChan}
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
	connectFrame := NewFrame(_COMMAND_CONNECT, headers, _NULLBUFF,client.connectionErr)
	client.writeFrame(connectFrame)
	//b, e := client.reader.ReadBytes(0)
	//if e != nil {
	//	fmt.Println("returned from connect ", string(b))
	//	return e
	//}
	fmt.Println("reading after connect")
	f,err := client.readFrame()
	if err != nil{
		return err
	}
	fmt.Print("read frame ",f)
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
			if _,ok := err.(ConnectionError); ok {
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

//reads a single frame of the wire
func (client *Client) readFrame()(Frame,error){
	f := Frame{}
	line, err := client.reader.ReadBytes('\n')
	//count this as a connection error
	if err != nil {
		return f,ConnectionError(err.Error())
	}
	fmt.Println("read line ", string(line))
	f.Command = line

	f.Headers = make(map[string]string)
	for{
		header, err := client.reader.ReadString('\n')
		if nil != err{
			fmt.Println("error reading headers ",err,header)
			return f,err
		}
		if header == "\n"{
			//reached end of headers break
			break
		}
		parsed := strings.SplitN(header,":",2)
		if len(parsed) !=2{
			//this is an error handle
			fmt.Println("error ", parsed)
		}
		//todo need to decode the headers
		f.Headers[parsed[0]] = parsed[1]


		fmt.Println(header)

	}
	//read headers
	//line, err = client.reader.ReadString('\n')
	//if err != nil {
	//	return f,err
	//}
	//fmt.Println("read line ", line)
	//ready body
	body,err := client.reader.ReadBytes('\n')
	if err != nil {
		return f,ConnectionError(err.Error())
	}
	fmt.Println("body of message ", string(body))
	//if we are connecting and there was an error let the client know
	if client._connecting{
		if string(f.Command) == "ERROR\n"{
			//look for a message header to get more info
			return f, ServerError("error during initial connection " + f.Headers["message"])
			fmt.Println("error returned from server")
		}
	}
	return f,nil
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
