package stompy

import (
	"testing"
	"time"

	"fmt"
	"os"

	"github.com/maleck13/stompy/Godeps/_workspace/src/github.com/stretchr/testify/assert"
)

var (
	SKIP_INTEGRATION   = os.Getenv("TEST_SKIP_INTEGRATION")
	INTEGRATION_SERVER = os.Getenv("INTEGRATION_SERVER")
)

func TestClient_Connect_Error(t *testing.T) {
	opts := ClientOpts{
		HostAndPort: "idonetexist:61613",
		Timeout:     100 * time.Millisecond,
		Vhost:       "localhost",
	}
	client := NewClient(opts)
	errorReceived := false
	client.RegisterDisconnectHandler(func(err error) {
		fmt.Println("recieved disconnect err ", err)
		errorReceived = true
	})
	if err := client.Connect(); err != nil {
		assert.Error(t, err, "expected a connection error")
	}
	time.Sleep(20 * time.Millisecond) //give it time to receive the channel msg
	assert.True(t, errorReceived, "expected an error to be recieved")
}

func TestClient_Connect_Ok(t *testing.T) {
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

func TestClient_Connect_NotOkBadAuth(t *testing.T) {
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

func TestClient_Connect_NotOkBadHost(t *testing.T) {
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

func TestClient_PublishBasicSend(t *testing.T) {
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
	assert.NoError(t, err, "did not expect a connection error ")
	//defer client.Disconnect()
	err = client.Publish("/test/test", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, nil)
	assert.NoError(t, err, "did not expect a connection error ")
	time.Sleep(1000 * time.Millisecond) //give it time to receive the channel msg

}

func TestClient_Subscribe(t *testing.T) {
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
	assert.NoError(t, err, "did not expect a connection error ")
	err = client.Subscribe("/test/test", func(f Frame) {
		fmt.Println("recieved message ", string(f.Body))
	}, StompHeaders{},nil)
	assert.NoError(t, err, "did not expect an error subscribing ")
	for i := 0; i < 20; i++ {
		str := fmt.Sprintf("test %d ", i)
		err = client.Publish("/test/test", "application/json", []byte(`{"test":"`+str+`"}`), StompHeaders{}, nil)
	}
	assert.NoError(t, err, "did not expect an error subscribing ")
	time.Sleep(500 * time.Millisecond) //give it time to receive the channel msg
}

func TestClient_SubscribeWithReceipt(t *testing.T) {
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
	assert.NoError(t, err, "did not expect a connection error ")
	rec := NewReceipt(time.Second * 1)
	err = client.Subscribe("/test/test", func(f Frame) {
		fmt.Println("recieved message ", string(f.Body))
	}, StompHeaders{},rec)
	received := <- rec.receiptReceived
	assert.NoError(t, err, "did not expect an error subscribing ")
	for i := 0; i < 20; i++ {
		str := fmt.Sprintf("test %d ", i)
		err = client.Publish("/test/test", "application/json", []byte(`{"test":"`+str+`"}`), StompHeaders{}, nil)
	}
	assert.NoError(t, err, "did not expect an error subscribing ")
	assert.True(t, received, "expected a receipt")
}

func TestClient_PublishWithReceipt(t *testing.T) {
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
	assert.NoError(t, err, "did not expect a connection error ")
	err = client.Subscribe("/test/test", func(f Frame) {
		fmt.Println("recieved message ", string(f.Body))
	}, StompHeaders{},nil)
	assert.NoError(t, err, "did not expect an error subscribing ")
	headers := StompHeaders{}
	headers["receipt"] = "message-1"
	rec := NewReceipt(time.Second * 1)
	err = client.Publish("/test/test", "application/json", []byte(`{"test":"test"}`), headers, rec)
	assert.NoError(t, err, "did not expect an error subscribing ")
	received := <-rec.receiptReceived
	assert.True(t, received, "expected a receipt")
}
