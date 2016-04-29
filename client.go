//Stompy is a stomp client for communicating with stomp based messaging servers.
//It supports stomp 1.1 and 1.2. It exposes a set of interfaces to allow easy mocking
// in tests. The main interface is StompClient

package stompy

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

//Supported Versions of stomp protocol
const (
	STOMP_1_1 string = "1.1"
	STOMP_1_2 string = "1.2"
)

var (
	Supported = []string{STOMP_1_1, STOMP_1_2}
	//A dumb default Disconnect Handler
	DefaultDisconnectHandler = func(err error) {
		//hmmm what to do
		log.Println("defualt disconnect handler: ", err)
	}
)

//Available connection and auth params
type ClientOpts struct {
	Vhost       string
	HostAndPort string
	Timeout     time.Duration
	User        string
	PassCode    string
	Version     string
}

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
	Subscribe(destination string, handler SubscriptionHandler, headers StompHeaders, receipt *Receipt) (string, error)
	Unsubscribe(subId string, headers StompHeaders, receipt *Receipt) error
	Ack(msg Frame) error
	Nack(msg Frame) error
}

//responsible for defining how a publish should happen
type StompPublisher interface {
	//accepts a body, destination, content-type and any headers you wish to override or set
	Publish(destination string, contentType string, body []byte, headers StompHeaders, receipt *Receipt) error
}

//defines how transactions are done
type StompTransactor interface {
	Begin(transId string, addedHeaders StompHeaders, receipt *Receipt) error
	Abort(transId string, addedHeaders StompHeaders, receipt *Receipt) error
	Commit(transId string, addedHeaders StompHeaders, receipt *Receipt) error
}

//A stomp client is a combination of all of these things
type StompClient interface {
	StompConnector
	StompSubscriber
	StompPublisher
	StompTransactor
}

type messageStats struct {
	sync.Mutex
	count int
}

func (s *messageStats) Increment() int {
	s.Lock()
	defer s.Unlock()
	s.count++
	return s.count
}

var stats = &messageStats{}

//main client type for interacting with stomp. This is the exposed type
type Client struct {
	opts              ClientOpts
	connectionErr     chan error        //we send an error on this channel if there is a connection error. the DisconnectHandler is called if this channel receives an error
	shutdown          chan bool         // tell any loops to exit as we are disconnecting. For example the readLoop
	msgChan           chan Frame        // read loop sends new messages on this channel
	DisconnectHandler DisconnectHandler // a func that should do something in the case of a network disconnection
	conn              net.Conn
	reader            stompSocketReader //used to read from the network socket
	subscriptions     *subscriptions
	encoderDecoder    encoderDecoder
	headersFactory    headers
}

//Create a new stomp client based on a set of options
func NewClient(opts ClientOpts) StompClient {
	errChan := make(chan error)
	shutdown := make(chan bool, 1)
	msgChan := make(chan Frame)
	subMap := make(map[string]subscription)
	subs := &subscriptions{subs: subMap}
	encoderDecoder := headerEncoderDecoder{opts.Version}
	headersFactory := headers{version: opts.Version}
	return &Client{opts: opts, connectionErr: errChan, shutdown: shutdown, subscriptions: subs,
		msgChan: msgChan, encoderDecoder: encoderDecoder, headersFactory: headersFactory}
}

//Begin a transaction with the stomp server
func (client *Client) Begin(transId string, addedHeaders StompHeaders, receipt *Receipt) error {
	headers := client.headersFactory.transactionHeaders(transId, addedHeaders)
	headers, err := handleReceipt(headers, receipt)
	if err != nil {
		return err
	}
	f := NewFrame(_COMAND_TRANSACTION_BEGIN, headers, _NULLBUFF)
	if err := writeFrame(bufio.NewWriter(client.conn), f, client.encoderDecoder); err != nil {
		return err
	}
	return nil
}

//Abort a transaction with the stomp server
func (client *Client) Abort(transId string, addedHeaders StompHeaders, receipt *Receipt) error {
	headers := client.headersFactory.transactionHeaders(transId, addedHeaders)
	headers, err := handleReceipt(headers, receipt)
	if err != nil {
		return err
	}
	f := NewFrame(_COMAND_TRANSACTION_ABORT, headers, _NULLBUFF)
	if err := writeFrame(bufio.NewWriter(client.conn), f, client.encoderDecoder); err != nil {
		return err
	}
	return nil
}

//Commit a transaction with the stomp server
func (client *Client) Commit(transId string, addedHeaders StompHeaders, receipt *Receipt) error {
	headers := client.headersFactory.transactionHeaders(transId, addedHeaders)
	headers, err := handleReceipt(headers, receipt)
	if err != nil {
		return err
	}
	f := NewFrame(_COMAND_TRANSACTION_COMMIT, headers, _NULLBUFF)
	if err := writeFrame(bufio.NewWriter(client.conn), f, client.encoderDecoder); err != nil {
		return err
	}
	return nil
}

//Acknowledge receipt of a message with stomp server
func (client *Client) Ack(msg Frame) error {
	if _, ok := msg.Headers["message-id"]; !ok {
		return ClientError("cannot ack message without message-id header")
	}
	msgId := msg.Headers["message-id"]
	transId := msg.Headers["transaction"]
	subId := msg.Headers["subscription"]
	ackid := msg.Headers["ack"]
	headers := client.headersFactory.ackHeaders(msgId, subId, ackid, transId)
	frame := NewFrame(_COMMAND_ACK, headers, _NULLBUFF)
	return writeFrame(bufio.NewWriter(client.conn), frame, client.encoderDecoder)
}

//Dont acknowledge the message and let the server know so it can decide what to do with it
func (client *Client) Nack(msg Frame) error {
	if _, ok := msg.Headers["message-id"]; !ok {
		return ClientError("cannot ack message without message-id header")
	}
	msgId := msg.Headers["message-id"]
	transId := msg.Headers["transaction"]
	subId := msg.Headers["subscription"]
	ackid := msg.Headers["ack"]
	headers := client.headersFactory.nackHeaders(msgId, subId, ackid, transId)
	frame := NewFrame(_COMMAND_NACK, headers, _NULLBUFF)
	return writeFrame(bufio.NewWriter(client.conn), frame, client.encoderDecoder)
}

//StompConnector.Connect creates a tcp connection. sends any error through the errChan also returns the error
func (client *Client) Connect() error {
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

	//setup version specific header encoder
	encoder := headerEncoderDecoder{client.opts.Version}

	client.reader = newStompReader(conn, client.shutdown, client.connectionErr, client.msgChan, encoder)

	headers, err := client.headersFactory.connectionHeaders(client.opts)
	if err != nil {
		return ConnectionError(err.Error())
	}
	connectFrame := NewFrame(_COMMAND_CONNECT, headers, _NULLBUFF)
	if err := writeFrame(bufio.NewWriter(conn), connectFrame, encoder); err != nil {
		client.sendConnectionError(err)
		return err
	}

	//read frame after writing out connection to check we are connected
	f, err := client.reader.readFrame()
	if err != nil {
		client.sendConnectionError(err)
		return err
	}

	if f.CommandString() == "ERROR" {
		return ClientError("after initial connection recieved err : " + f.Headers["message"])
	}
	if err := versionCheck(f); err != nil {
		return err
	}
	//start background readloop
	go client.reader.startReadLoop()
	//start background dispatch
	go client.subscriptions.dispatch(client.msgChan)

	return nil

}

//StompConnector.Disconnect close our error channel then close the socket connection
func (client *Client) Disconnect() error {
	if nil != client.conn {
		headers := StompHeaders{}
		headers["receipt"] = "disconnect"
		rec := NewReceipt(time.Second * 1)
		awaitingReceipt.Add("disconnect", rec)
		frame := NewFrame(_COMMAND_DISCONNECT, headers, _NULLBUFF)

		if err := writeFrame(bufio.NewWriter(client.conn), frame, client.encoderDecoder); err != nil {
			return err
		}
		<-rec.Received
		//signal read loop to shutdown
		client.shutdown <- true
		client.subscriptions = newSubscriptions()
		close(client.connectionErr)
		close(client.shutdown)
		close(client.msgChan)
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
func (client *Client) Publish(destination, contentType string, body []byte, addedHeaders StompHeaders, receipt *Receipt) error {

	headers := client.headersFactory.sendHeaders(destination, contentType, addedHeaders)
	headers, err := handleReceipt(headers, receipt)
	if err != nil {
		return err
	}

	frame := NewFrame(_COMMAND_SEND, headers, body)

	return writeFrame(bufio.NewWriter(client.conn), frame, client.encoderDecoder)
}

//subscribe to messages sent to the destination. The SubscriptionHandler will also receive RECEIPTS if a receipt header is set
//headers are id and ack
func (client *Client) Subscribe(destination string, handler SubscriptionHandler, headers StompHeaders, receipt *Receipt) (string, error) {
	//create an id
	//ensure we don't end up with double registration
	sub, err := newSubscription(destination, handler, headers)
	if nil != err {
		return "", err
	}
	if err := client.subscriptions.addSubscription(sub); err != nil {
		return "", err
	}
	subHeaders := client.headersFactory.subscribeHeaders(sub.Id, destination, headers)
	subHeaders, err = handleReceipt(subHeaders, receipt)
	if err != nil {
		return "", err
	}
	frame := Frame{_COMMAND_SUBSCRIBE, subHeaders, _NULLBUFF}
	//todo think about if we have no conn
	if err := writeFrame(bufio.NewWriter(client.conn), frame, client.encoderDecoder); err != nil {
		return "", err
	}
	return sub.Id, nil
}

//Unsubscribe takes the id of a subscription and removes that subscriber so it will no longer receive messages
func (client *Client) Unsubscribe(id string, headers StompHeaders, receipt *Receipt) error {
	unSub := client.headersFactory.unSubscribeHeaders(id, headers)
	unSub, err := handleReceipt(unSub, receipt)
	if err != nil {
		return err
	}
	frame := Frame{_COMMAND_UNSUBSCRIBE, unSub, _NULLBUFF}
	client.subscriptions.removeSubscription(id)
	if err := writeFrame(bufio.NewWriter(client.conn), frame, client.encoderDecoder); err != nil {
		return err
	}
	return nil
}

func versionCheck(f Frame) error {
	var ok = false
	version := f.Headers["version"]
	if "" != version {
		for _, v := range Supported {
			if v == version {
				ok = true
				break
			}
		}
	}
	if !ok {
		return VersionError("unsupported version " + version)
	}
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

func handleReceipt(headers StompHeaders, receipt *Receipt) (StompHeaders, error) {
	if receiptId, ok := headers["receipt"]; ok && receipt != nil {
		if err := awaitingReceipt.Add(receiptId, receipt); err != nil {
			return headers, err
		}
	} else if nil != receipt {
		receiptId := "message-" + strconv.Itoa(stats.Increment())
		headers["receipt"] = receiptId
		awaitingReceipt.Add(receiptId, receipt)
	}

	return headers, nil
}
