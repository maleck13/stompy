package stompy

import (
	"testing"
	"time"

	"fmt"
	"os"

	"sync"

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
	fmt.Println("after connect in test", err)
	assert.Error(t, err, "expect an error ")
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
	wait := &sync.WaitGroup{}
	_, err = client.Subscribe("/test/testsub", func(f Frame) {
		wait.Done()
		fmt.Println("recieved message ", string(f.Body))
	}, StompHeaders{}, nil)
	assert.NoError(t, err, "did not expect an error subscribing ")

	for i := 0; i < 20; i++ {
		wait.Add(1)
		str := fmt.Sprintf("test %d ", i)
		err = client.Publish("/test/testsub", "application/json", []byte(`{"test":"`+str+`"}`), StompHeaders{}, nil)
	}
	wait.Wait()
	client.Disconnect()
	assert.NoError(t, err, "did not expect an error subscribing ")
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
	_, err = client.Subscribe("/test/test", func(f Frame) {
		fmt.Println("recieved message ", string(f.Body))
	}, StompHeaders{}, rec)
	received := <-rec.receiptReceived
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
	_, err = client.Subscribe("/test/test", func(f Frame) {
		//fmt.Println("recieved message ", string(f.Body))
	}, StompHeaders{}, nil)
	client.RegisterDisconnectHandler(func(err error) {
		fmt.Println("recieved disconnect err ", err)
		assert.Fail(t, err.Error(), "disconnect error")
	})
	assert.NoError(t, err, "did not expect an error subscribing ")
	wait := &sync.WaitGroup{}
	for i := 0; i < 200; i++ {
		go func() {
			wait.Add(1)
			rec := NewReceipt(time.Second * 2)
			err = client.Publish("/test/test", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, rec)
			assert.NoError(t, err, "did not expect an error publishing ")
			received := <-rec.Received
			assert.True(t, received, "expected a receipt")
			wait.Done()
		}()
	}
	wait.Wait()

}

func TestClient_Disconnect(t *testing.T) {
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
	err = client.Disconnect()
	assert.NoError(t, err, "did not expect a disconnect error ")
}

func TestClient_Connect_BadVers(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	opts := ClientOpts{
		HostAndPort: INTEGRATION_SERVER,
		Timeout:     20 * time.Second,
		Vhost:       "localhost",
		User:        "admin",
		PassCode:    "admin",
		Version:     "1.0",
	}
	client := NewClient(opts)
	err := client.Connect()
	fmt.Println(err)
	assert.Error(t, err, " expected an error connectiong with unsupported version")
}

func TestClient_Unsubscribe(t *testing.T) {
	//if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
	//	t.Skip("INTEGRATION DISABLED")
	//}
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
	assert.NoError(t, err, "did not expect error connecting")
	var rec = false
	var wait = &sync.WaitGroup{}
	id, err := client.Subscribe("/test/unsub", func(f Frame) {
		fmt.Println(f.CommandString())
		assert.False(t, rec, "should only recieve one message")
		rec = true
		wait.Done()
	}, StompHeaders{}, nil)
	fmt.Println("sub id is ", id)
	assert.NoError(t, err, "did not expect an error subscribing")
	assert.NotEqual(t, "", id, "expected a subscription id")
	wait.Add(1)
	err = client.Publish("/test/unsub", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, nil)
	assert.NoError(t, err, "did not expect error publishing")
	wait.Wait()
	receipt := NewReceipt(time.Millisecond * 100)
	err = client.Unsubscribe(id, StompHeaders{}, receipt)
	<-receipt.Received
	assert.NoError(t, err, "did not expect error unsub")
	err = client.Publish("/test/unsub", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, nil)
	time.Sleep(time.Millisecond * 200)

}

func TestClient_Publish_Client_Ack_client(t *testing.T) {
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
	assert.NoError(t, err, "did not expect error connecting")

	//simulate two subscriptions
	var wait = &sync.WaitGroup{}
	headers := StompHeaders{}
	headers["ack"] = "client"
	wait.Add(1)
	rec := NewReceipt(time.Second * 2)

	id, err := client.Subscribe("/test/ack", func(f Frame) {
		fmt.Println("sub 1  **** ", f.CommandString())
		wait.Done()
	}, headers, nil)
	err = client.Publish("/test/ack", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, rec)
	assert.NoError(t, err, "did not expect an error publishing ")

	<-rec.Received
	wait.Wait()
	client.Unsubscribe(id, StompHeaders{}, nil)

	wait.Add(1)
	_, err = client.Subscribe("/test/ack", func(f Frame) {
		fmt.Println("sub 2  **** ", f.CommandString())
		client.Ack(f)
		wait.Done()
	}, headers, nil)

	assert.NoError(t, err, "did not expect error subscribing")
	fmt.Println("waiting")
	wait.Wait()

}

func TestClient_Publish_Client_Ack_client_individual(t *testing.T) {
	opts := ClientOpts{
		HostAndPort: INTEGRATION_SERVER,
		Timeout:     20 * time.Second,
		Vhost:       "localhost",
		User:        "admin",
		PassCode:    "admin",
		Version:     "1.2",
	}
	client := NewClient(opts)
	err := client.Connect()
	assert.NoError(t, err, "did not expect error connecting")

	//simulate two subscriptions
	var wait = &sync.WaitGroup{}
	headers := StompHeaders{}
	headers["ack"] = "client-individual"
	wait.Add(1)
	_, err = client.Subscribe("/test/ack2", func(f Frame) {
		if err := client.Ack(f); err != nil {
			assert.NoError(t, err, "did not expect err")
		}
		wait.Done()
	}, headers, nil)

	sendHeaders := StompHeaders{}
	//sendHeaders["persistent"] = "true"
	//expire := time.Now().Add(time.Second*2).UTC().Unix() * 1000
	//sendHeaders["expires"] = strconv.Itoa(int(expire))
	err = client.Publish("/test/ack2", "application/json", []byte(`{"test":"test"}`), sendHeaders, nil)
	assert.NoError(t, err, "did not expect an error publishing ")
	wait.Wait()
}
