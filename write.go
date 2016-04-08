package stompy

import (
	"strconv"
	"io"
)

type SocketWriter interface {
	io.Writer
	WriteByte(c byte)error
	Flush()error
}

func writeFrame(writer SocketWriter, frame Frame) error {

	//set content length
	frame.Headers["content-length"] = strconv.Itoa(len(frame.Body))
	//write our command CONNECT SUBSCRIBE etc to the buffer
	_, err := writer.Write(frame.Command)
	if err != nil {
		//treating failure to write to the socket as a network error
		return err
	}

	//write each of our headers to the buffer
	for k, v := range frame.Headers {
		val := k + ":" + v + "\n"
		if _, err := writer.Write([]byte(val)); err != nil {
			return err
		}
	}
	//as per the spec add a new line after the header to the buffer
	if err := writer.WriteByte('\n'); err != nil {
		return err
	}
	//write our body if there is one to the buffer
	if len(frame.Body) > 0 {
		if _, err := writer.Write(frame.Body); err != nil {
			return err
		}
	}
	//when we flush the buffer to the socket then if there is an error it is a network error
	if err := writer.Flush(); err != nil {
		return ConnectionError(err.Error())
	}
	//stomp protocol want a null byte at the end of the frame
	if err := writer.WriteByte('\x00'); err != nil {
		return err
	}

	//again when we flush the buffer to the socket then if there is an error it is a network error
	if err := writer.Flush(); err != nil {
		return ConnectionError(err.Error())
	}
	return nil
}
