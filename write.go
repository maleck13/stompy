package stompy

import (
	"bufio"
	"strconv"
)

func writeFrame(writer *bufio.Writer, frame Frame) error {

	//set content length
	frame.Headers["content-length"] = strconv.Itoa(len(frame.Body))
	//write our command CONNECT SUBSCRIBE etc
	if _, err := writer.Write(frame.Command); err != nil {
		//treating failure to write to the socket as a network error
		return ConnectionError(err.Error())
	}

	//write each of our headers
	for k, v := range frame.Headers {
		val := k + ":" + v + "\n"
		if _, err := writer.Write([]byte(val)); err != nil {
			return err
		}
	}
	//as per the spec add a new line after the header
	if err := writer.WriteByte('\n'); err != nil {
		return err
	}
	//write our body if there is one
	if len(frame.Body) > 0 {
		if _, err := writer.Write(frame.Body); err != nil {
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	//stomp protocol want a null byte at the end of the frame
	if err := writer.WriteByte('\x00'); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	return nil
}
