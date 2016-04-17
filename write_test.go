package stompy

import (
	"errors"
	"testing"

	"github.com/maleck13/stompy/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

//implement SocketWriter
type MockStompWriter struct {
	writeError     error
	flushError     error
	byteWriteError error
}

func (m *MockStompWriter) Write(p []byte) (n int, err error) {

	return len(p), m.writeError
}

func (m *MockStompWriter) WriteByte(c byte) error {
	return m.byteWriteError
}

func (m *MockStompWriter) Flush() error {
	return m.flushError
}

func Test_write_frame_ok(t *testing.T) {
	encoder := headerEncoderDecoder{"1.1"}
	writer := &MockStompWriter{}
	frame := Frame{Command: _COMMAND_CONNECT, Headers: StompHeaders{}, Body: _NULLBUFF}
	err := writeFrame(writer, frame,encoder)
	assert.NoError(t, err, "did not expect an error writing")
}

func Test_write_frame_err(t *testing.T) {
	encoder := headerEncoderDecoder{"1.1"}
	mockWriter := &MockStompWriter{flushError: errors.New("unexpected")}
	frame := Frame{Command: _COMMAND_CONNECT, Headers: StompHeaders{}, Body: _NULLBUFF}
	err := writeFrame(mockWriter, frame,encoder)
	assert.Error(t, err, "expected an error to be returned when writing failed")
	if _, ok := err.(ConnectionError); !ok {
		assert.Fail(t, "error should be a connection Error")
	}
}
