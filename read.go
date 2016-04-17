package stompy

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

type StompReader interface {
	ReadBytes(c byte) ([]byte, error)
	ReadString(c byte) (string, error)
}

type StompSocketReader struct {
	decoder  decoder
	reader   StompReader
	shutdown chan bool
	errChan  chan error
	msgChan  chan Frame // we may be over complicating here
}

func NewStompReader(con net.Conn, shutdownCh chan bool, errChan chan error, msgChan chan Frame, decoder decoder) StompSocketReader {
	return StompSocketReader{
		decoder:  decoder,
		reader:   bufio.NewReader(con),
		shutdown: shutdownCh,
		errChan:  errChan,
		msgChan:  msgChan,
	}
}

//reads a single frame of the wire
func (sr StompSocketReader) readFrame() (Frame, error) {
	f := Frame{}

	line, err := sr.reader.ReadBytes('\n')
	//count this as a connection error. will be sent via error channel to the reconnect handler
	if err != nil {
		return f, ConnectionError(err.Error())
	}
	f.Command = line
	//sort out our headers
	f.Headers = StompHeaders{}
	for {
		header, err := sr.reader.ReadString('\n')
		if nil != err {
			return f, err
		}
		if header == "\n" {
			//reached end of headers break should we set some short deadlock break?
			break
		}

		parsed := strings.SplitN(header[0:len(header)-1], ":", 2)
		if len(parsed) != 2 {
			return f, BadFrameError("failed to parse header correctly " + header)
		}
		//todo need to decode the headers
		key := sr.decoder.Decode(parsed[0])
		val := sr.decoder.Decode(parsed[1])
		f.Headers[key] = val
	}
	//ready body
	body, err := sr.reader.ReadBytes('\n')
	if err != nil {
		return f, err
	}
	f.Body = body[0 : len(body)-1]

	return f, nil
}

func (sr StompSocketReader) startReadLoop() {
	//read a frame, if it is a subscription send it be handled
	for {
		frame, err := sr.readFrame() //this will block until it reads
		//if we have clean shutdown ignore the error as the connection has been closed
		select {
		case <-sr.shutdown:
			fmt.Println("error reading ", err, "ignoring as shutdown received")
			return
		default:

			if err != nil {
				sr.errChan <- ConnectionError("failed when reading frame " + err.Error())
				sr.shutdown <- true
			} else {
				sr.msgChan <- frame
			}
		}

	}

}
