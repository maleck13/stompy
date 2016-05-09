package stompy

import (
	"testing"

	"bufio"
	"strings"

	"github.com/stretchr/testify/assert"
)

//this test is a bit fragile. Look at improving
func TestReadFrameOK(t *testing.T) {
	r := strings.NewReader("MESSAGE\ntest:test\n\n{\"test\":\"test\"}\n")
	reader := bufio.NewReader(r)
	shutdown := make(chan bool)
	errChan := make(chan error)
	msgChan := make(chan Frame)
	socketReader := stompSocketReader{
		decoder:  headerEncoderDecoder{},
		reader:   reader,
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
	r := strings.NewReader("MESSAGE\ntesttest\n\n{\"test\":\"test\"}\n")
	reader := bufio.NewReader(r)
	shutdown := make(chan bool)
	errChan := make(chan error)
	msgChan := make(chan Frame)
	socketReader := stompSocketReader{
		decoder:  headerEncoderDecoder{},
		reader:   reader,
		shutdown: shutdown,
		errChan:  errChan,
		msgChan:  msgChan,
	}

	frame, err := socketReader.readFrame()
	assert.Error(t, err, "expected an error reading")
	assert.NotNil(t, frame, " expected a frame")
}
