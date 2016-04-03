package stompy

var (
	_COMMAND_CONNECT    []byte = []byte("CONNECT\n")
	_COMMAND_DISCONNECT []byte = []byte("DISCONNECT\n")
	_COMMAND_SUBSCRIBE  []byte = []byte("SUBSCRIBE\n")
	_COMMAND_SEND  []byte = []byte("SEND\n")
	_NULLBUFF                  = make([]uint8, 0)
	newline    = byte(10)
	cr         = byte(13)
	colon      = byte(58)
	nullByte   = byte(0)
)

//stomp frame is made up of a command, headers and body. The err channel is for communicating back to the client on connection error
type Frame struct {
	Command   []byte
	Headers   map[string]string
	Body      []byte
	errorChan chan error
}

func NewFrame(command []byte, headers HEADERS, body []byte, errChan chan error) Frame {
	return Frame{
		Command: command,
		Headers: headers,
		Body:    body,
		errorChan: errChan,
	}
}
