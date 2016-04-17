package stompy

var (
	_COMMAND_CONNECT           []byte = []byte("CONNECT\n")
	_COMMAND_DISCONNECT        []byte = []byte("DISCONNECT\n")
	_COMMAND_SUBSCRIBE         []byte = []byte("SUBSCRIBE\n")
	_COMMAND_UNSUBSCRIBE       []byte = []byte("UNSUBSCRIBE\n")
	_COMMAND_SEND              []byte = []byte("SEND\n")
	_COMMAND_ACK               []byte = []byte("ACK\n")
	_COMMAND_NACK              []byte = []byte("NACK\n")
	_COMAND_TRANSACTION_BEGIN  []byte = []byte("BEGIN\n")
	_COMAND_TRANSACTION_COMMIT []byte = []byte("COMMIT\n")
	_COMAND_TRANSACTION_ABORT  []byte = []byte("ABORT\n")
	_NULLBUFF                         = make([]uint8, 0)
	newline                           = byte(10)
	cr                                = byte(13)
	colon                             = byte(58)
)

//stomp frame is made up of a command, headers and body. The err channel is for communicating back to the client on connection error
type Frame struct {
	Command []byte
	Headers map[string]string
	Body    []byte
}

func (f Frame) CommandString() string {
	if len(f.Command) > 0 {
		return string(f.Command[0 : len(f.Command)-1])
	}
	return ""
}

func NewFrame(command []byte, headers StompHeaders, body []byte) Frame {
	return Frame{
		Command: command,
		Headers: headers,
		Body:    body,
	}
}
