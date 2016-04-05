package stompy

import (
	"testing"
	"time"

	"fmt"
	"os"

	"github.com/stretchr/testify/assert"
)

var (
	SKIP_INTEGRATION   = os.Getenv("TEST_SKIP_INTEGRATION")
	INTEGRATION_SERVER = os.Getenv("INTEGRATION_SERVER")
)

func TestConnectionError(t *testing.T) {
	opts := ClientOpts{
		HostAndPort: "idonetexist:61613",
		Timeout:     100 * time.Millisecond,
		Vhost:       "localhost",
	}
	client := NewClient(opts)
	errorReceived := false
	err := client.RegisterDisconnectHandler(func(err error) {
		fmt.Println("recieved disconnect err ", err)
		errorReceived = true
	})
	assert.NoError(t, err, "did not expect an error registering disconnect handler")
	if err := client.Connect(); err != nil {
		assert.Error(t, err, "expected a connection error")
	}
	time.Sleep(20 * time.Millisecond) //give it time to receive the channel msg
	assert.True(t, errorReceived, "expected an error to be recieved")
	client.Disconnect()
}

func TestConnectionOk(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	opts := ClientOpts{
		HostAndPort: INTEGRATION_SERVER,
		Timeout:     20 * time.Second,
		Vhost:       "localhost",
		User:        "admin",
		PassCode:    "admin",
		Version:     "1.1",
	}
	client := NewClient(opts)
	err := client.Connect()
	fmt.Println("after connect in test")
	assert.NoError(t, err, "did not expect a connection error ")
	client.Disconnect()
}

func TestConnectionNotOkBadAuth(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	opts := ClientOpts{
		HostAndPort: INTEGRATION_SERVER,
		Timeout:     20 * time.Second,
		Vhost:       "localhost",
		User:        "admin",
		PassCode:    "badpass",
		Version:     "1.1",
	}
	client := NewClient(opts)
	err := client.Connect()
	fmt.Println("after connect in test")
	assert.NoError(t, err, "did not expect a connection error ")
	client.Disconnect()
}


func TestConnectionNotOkBadHost(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	opts := ClientOpts{
		HostAndPort: "localhost",
		Timeout:     20 * time.Second,
		Vhost:       "localhost",
		User:        "admin",
		PassCode:    "badpass",
		Version:     "1.1",
	}
	client := NewClient(opts)
	err := client.Connect()
	fmt.Println("after connect in test")
	assert.Error(t, err, "expected a connection error ")
	client.Disconnect()
}


