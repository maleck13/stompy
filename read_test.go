package stompy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockStompReader struct {
	readBytes  int
	readString int
}

func (mr *MockStompReader) ReadBytes(c byte) ([]byte, error) {
	var cmd = []byte("MESSAGE\n")
	var body = []byte(`{"test":"test"}`)
	var frame = make([][]byte, 0)
	frame = append(frame, cmd, body)
	data := frame[mr.readBytes]
	mr.readBytes++
	return data, nil
}

func (mr *MockStompReader) ReadString(c byte) (string, error) {
	if mr.readString == 0 {
		mr.readString++
		return "test:test\n", nil
	} else {
		return "\n", nil
	}
}

//this test is a bit fragile. Look at improving
func TestReadFrameOK(t *testing.T) {

	shutdown := make(chan bool)
	errChan := make(chan error)
	msgChan := make(chan Frame)
	socketReader := stompSocketReader{
		decoder:  headerEncoderDecoder{},
		reader:   &MockStompReader{},
		shutdown: shutdown,
		errChan:  errChan,
		msgChan:  msgChan,
	}

	frame, err := socketReader.readFrame()
	assert.NoError(t, err, "did not expect an error reading")
	assert.NotNil(t, frame, "expected a frame")
	assert.Equal(t, frame.CommandString(), "MESSAGE", "command should have been message")
	if _, k := frame.Headers["test"]; !k {
		assert.Fail(t, "exected the test header to be there")
	}
	assert.NotNil(t, frame.Body, "body should be present")
	assert.True(t, len(frame.Body) > 0, "body should have content")

}

func TestReadFrameError(t *testing.T) {

}
