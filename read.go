package stompy

import (
	"bufio"
	"net"
	"strings"
)

type stompReader interface {
	ReadBytes(c byte) ([]byte, error)
	ReadString(c byte) (string, error)
}

type stompSocketReader struct {
	decoder  decoder
	reader   stompReader
	shutdown chan bool
	errChan  chan error
	msgChan  chan Frame // we may be over complicating here
}

func newStompReader(con net.Conn, shutdownCh chan bool, errChan chan error, msgChan chan Frame, decoder decoder) stompSocketReader {
	return stompSocketReader{
		decoder:  decoder,
		reader:   bufio.NewReader(con),
		shutdown: shutdownCh,
		errChan:  errChan,
		msgChan:  msgChan,
	}
}

//reads a single frame of the wire
func (sr stompSocketReader) readFrame() (Frame, error) {
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
		key := sr.decoder.Decode(parsed[0])
		val := sr.decoder.Decode(parsed[1])
		f.Headers[key] = val
	}
	//read body
	body, err := sr.reader.ReadBytes('\n')
	if err != nil {
		return f, err
	}
	if 0 != len(body) {
		//return all but last 2 bytes which are a nul byte and a \n
		f.Body = body[0 : len(body)-2]
	}

	return f, nil
}

func (sr stompSocketReader) startReadLoop() {
	//read a frame, if it is a subscription send it be handled
	for {
		frame, err := sr.readFrame() //this will block until it reads
		//if we have clean shutdown ignore the error as the connection has been closed
		select {
		case <-sr.shutdown:
			return
		default:

			if err != nil {
				if _, ok := err.(BadFrameError); ok {
					sr.errChan <- err
				} else {
					sr.errChan <- ConnectionError("failed when reading frame " + err.Error())
					sr.shutdown <- true
				}
			} else {
				sr.msgChan <- frame
			}
		}

	}

}
