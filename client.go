package stompy

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
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
	Subscribe(destination string, handler SubscriptionHandler, headers StompHeaders, receipt *Receipt) error
}

//responsible for defining how a publish should happen
type StompPublisher interface {
	//accepts a body, destination, content-type and any headers you wish to override or set
	Publish(destination string, contentType string, body []byte, headers StompHeaders, receipt *Receipt) error
}

type StompTransactor interface {
	Begin(transId string, addedHeaders StompHeaders, receipt *Receipt) error
	Abort()
	Commit()
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

type Receipt struct {
	Received        chan bool
	receiptReceived chan bool
	Timeout         time.Duration
}

func (rec *Receipt) cleanUp(id string) {
	awaitingReceipt.Remove(id)
	close(rec.receiptReceived)
	close(rec.Received)
}

func NewReceipt(timeout time.Duration) *Receipt {
	return &Receipt{make(chan bool, 1), make(chan bool, 1), timeout}
}

type receipts struct {
	sync.RWMutex
	receipts map[string]*Receipt
}

func (r *receipts) Add(id string, rec *Receipt) error {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.receipts[id]; ok {
		return ClientError("already a receipt with that id " + id)
	}
	r.receipts[id] = rec
	//make sure we clean up. if we receive our receipt forward it on to the client
	//after the timeout send received as false.
	go func(receipt *Receipt, id string) {
		defer receipt.cleanUp(id)

		select {
		case <-rec.receiptReceived:
			fmt.Println("received reciept for ", id)
			rec.Received <- true
			break
		case <-time.After(rec.Timeout):
			fmt.Println("timeout exceeded for reciept ", id)
			rec.Received <- false
			break
		}

	}(rec, id)
	return nil
}

func (r *receipts) Remove(id string) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.receipts[id]; ok {
		delete(r.receipts, id)
		return
	}
	return
}

func (r *receipts) Count() int {
	r.RLock()
	defer r.RUnlock()
	return len(r.receipts)
}

func (r *receipts) Get(id string) *Receipt {
	r.RLock()
	defer r.RUnlock()
	return r.receipts[id]
}

var awaitingReceipt = &receipts{receipts: make(map[string]*Receipt)}

//main client type for interacting with stomp. This is the exposed type
type Client struct {
	opts              ClientOpts
	connectionErr     chan error        //we send an error on this channel if there is a connection error. the DisconnectHandler is called if this channel receives an error
	shutdown          chan bool         // tell any loops to exit as we are disconnecting. For example the readLoop
	msgChan           chan Frame        // read loop sends new messages on this channel
	DisconnectHandler DisconnectHandler // a func that should do something in the case of a network disconnection
	conn              net.Conn
	reader            StompSocketReader //used to read from the network socket
	subscriptions     *subscriptions
	sync.Mutex
}

//create a new client based on a set of options
func NewClient(opts ClientOpts) StompClient {
	errChan := make(chan error)
	shutdown := make(chan bool, 1)
	msgChan := make(chan Frame)
	subMap := make(map[string]subscription)
	subs := &subscriptions{subs: subMap}
	return &Client{opts: opts, connectionErr: errChan, shutdown: shutdown, subscriptions: subs, msgChan: msgChan}
}

func (client *Client) Begin(transId string, addedHeaders StompHeaders, receipt *Receipt) error {
	headers := transactionHeaders(transId, addedHeaders)
	headers, err := handleReceipt(headers, receipt)
	if err != nil {
		return err
	}
	f := NewFrame(_COMAND_TRANSACTION_BEGIN, headers, _NULLBUFF)
	if err := writeFrame(bufio.NewWriter(client.conn), f); err != nil {
		return err
	}
	return nil
}

func (client *Client) Abort() {

}

func (client *Client) Commit() {

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

	client.reader = NewStompReader(conn, client.shutdown, client.connectionErr, client.msgChan)

	headers, err := connectionHeaders(client.opts)
	if err != nil {
		return ConnectionError(err.Error())
	}
	connectFrame := NewFrame(_COMMAND_CONNECT, headers, _NULLBUFF)
	if err := writeFrame(bufio.NewWriter(conn), connectFrame); err != nil {
		fmt.Println("** error during initial connection write ** ", err)
		client.sendConnectionError(err)
		return err
	}

	//read frame after writing out connection to check we are connected
	if _, err = client.reader.readFrame(); err != nil {
		fmt.Println("** error during initial connection read **", err)
		client.sendConnectionError(err)
		return err
	}
	//start background readloop
	go client.reader.startReadLoop()
	//start background dispatch
	go client.subscriptions.dispatch(client.msgChan)

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
	//headers := StompHeaders{}
	//headers["receipt"] = "disconnect"
	//frame := NewFrame(_COMMAND_DISCONNECT,headers, _NULLBUFF)
	//err := writeFrame(client.writer,frame);
	//if nil == err {
	//	frame, err = client.reader.readFrame()
	//	fmt.Println("recieved final frame ",frame)
	//}

	//signal read loop to shutdown
	select {
	case client.shutdown <- true:
	default:
	}

	close(client.connectionErr)
	close(client.shutdown)
	close(client.msgChan)

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

//StompPublisher.Send publish a message to the server
func (client *Client) Publish(destination, contentType string, body []byte, addedHeaders StompHeaders, receipt *Receipt) error {

	headers := sendHeaders(destination, contentType, addedHeaders)
	headers, err := handleReceipt(headers, receipt)
	if err != nil {
		return err
	}

	frame := NewFrame(_COMMAND_SEND, headers, body)

	return writeFrame(bufio.NewWriter(client.conn), frame)
}

//subscribe to messages sent to the destination. The SubscriptionHandler will also receive RECEIPTS if a receipt header is set
//headers are id and ack
func (client *Client) Subscribe(destination string, handler SubscriptionHandler, headers StompHeaders, receipt *Receipt) error {
	//create an id
	//ensure we don't end up with double registration
	sub, err := NewSubscription(destination, handler, headers)
	if nil != err {
		return err
	}
	if err := client.subscriptions.addSubscription(sub); err != nil {
		return err
	}
	subHeaders := subscribeHeaders(sub.Id, destination, headers)
	subHeaders, err = handleReceipt(subHeaders, receipt)
	if err != nil {
		return err
	}
	frame := Frame{_COMMAND_SUBSCRIBE, subHeaders, _NULLBUFF}
	if err := writeFrame(bufio.NewWriter(client.conn), frame); err != nil {
		return err
	}
	return nil
}
