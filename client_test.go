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
	var tableOpts = []ClientOpts{
		GenerateClientOpts("idonetexist:61613", "admin", "admin", "1.1"),
		GenerateClientOpts("idonetexist:61613", "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		errorReceived := false
		wait := sync.WaitGroup{}
		wait.Add(1)
		client.RegisterDisconnectHandler(func(err error) {
			wait.Done()
			errorReceived = true
		})
		if err := client.Connect(); err != nil {
			assert.Error(t, err, "expected a connection error")
		}
		wait.Wait()
		assert.True(t, errorReceived, "expected an error to be recieved")
	}
}

func TestClient_Connect_Ok(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect a connection error ")
		client.Disconnect()
	}
}

func TestClient_Connect_NotOkBadAuth(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "badpass", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "badpass", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.Error(t, err, "expect an error ")
		client.Disconnect()
	}
}

func TestClient_Connect_NotOkBadHost(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts("localhost", "admin", "admin", "1.1"),
		GenerateClientOpts("localhost", "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.Error(t, err, "expected a connection error ")
		client.Disconnect()
	}
}

func TestClient_PublishBasicSend(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect a connection error ")
		rec := NewReceipt(time.Second * 1)
		err = client.Publish("/test/test", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, rec)
		assert.NoError(t, err, "did not expect a connection error ")
		received := <-rec.Received
		assert.True(t, received, "expected to receive a receipt after send")
	}

}

func TestClient_Subscribe(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect a connection error ")
		wait := &sync.WaitGroup{}
		_, err = client.Subscribe("/test/testsub", func(f Frame) {
			wait.Done()
			assert.Equal(t, "MESSAGE", f.CommandString(), "expected a message")
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
}

func TestClient_SubscribeWithReceipt(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect a connection error ")
		rec := NewReceipt(time.Second * 1)
		_, err = client.Subscribe("/test/test", func(f Frame) {
		}, StompHeaders{}, rec)
		received := <-rec.receiptReceived
		assert.NoError(t, err, "did not expect an error subscribing ")
		assert.True(t, received, "expected a receipt")
	}
}

func TestClient_PublishWithReceipt(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect a connection error ")
		wait := &sync.WaitGroup{}
		for i := 0; i < 200; i++ {
			go func() {
				wait.Add(1)
				rec := NewReceipt(time.Second * 1)
				err = client.Publish("/test/test", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, rec)
				assert.NoError(t, err, "did not expect an error publishing ")
				received := <-rec.Received
				assert.True(t, received, "expected a receipt")
				wait.Done()
			}()
		}
		wait.Wait()
	}
}

func TestClient_Disconnect(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect a connection error ")
		err = client.Disconnect()
		assert.NoError(t, err, "did not expect a disconnect error ")
	}
}

func TestClient_Connect_BadVersion(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	opts := GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.0")
	client := NewClient(opts)
	err := client.Connect()
	assert.Error(t, err, " expected an error connectiong with unsupported version")
}

func TestClient_UnSubscribe(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect error connecting")
		var rec = false
		var wait = &sync.WaitGroup{}
		id, err := client.Subscribe("/test/unsub", func(f Frame) {
			assert.False(t, rec, "should only recieve one message")
			rec = true
			wait.Done()
		}, StompHeaders{}, nil)
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
		time.Sleep(time.Millisecond * 100)
	}
}

func TestClient_Publish_Client_Ack_client(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect error connecting")

		//simulate two subscriptions
		var wait = &sync.WaitGroup{}
		headers := StompHeaders{}
		headers["ack"] = "client"
		wait.Add(1)
		rec := NewReceipt(time.Second * 1)
		id, err := client.Subscribe("/test/ack", func(f Frame) {
			wait.Done()
		}, headers, nil)
		err = client.Publish("/test/ack", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, rec)
		assert.NoError(t, err, "did not expect an error publishing ")
		<-rec.Received
		wait.Wait()
		rec = NewReceipt(time.Second * 1)
		client.Unsubscribe(id, StompHeaders{}, rec)
		<-rec.Received

		wait.Add(1)
		_, err = client.Subscribe("/test/ack", func(f Frame) {
			client.Ack(f)
			wait.Done()
		}, headers, nil)
		assert.NoError(t, err, "did not expect error subscribing")
		wait.Wait()
		err = client.Disconnect()
		assert.NoError(t, err, "did not expect error disconnecting")
	}
}

func TestClient_Publish_Client_Ack_client_individual(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
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
		err = client.Publish("/test/ack2", "application/json", []byte(`{"test":"test"}`), sendHeaders, nil)
		assert.NoError(t, err, "did not expect an error publishing ")
		wait.Wait()
		err = client.Disconnect()
		assert.NoError(t, err, "did not expect an error disconnecting")
	}
}

func TestClient_Transaction_Commit(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect error connecting")
		rec := NewReceipt(time.Second * 1)
		_, err = client.Subscribe("/test/trans", func(f Frame) {
			fmt.Println(f.Headers)
		}, StompHeaders{}, nil)

		err = client.Begin("transid", StompHeaders{}, rec)
		assert.NoError(t, err, "did not expect an error transaction Begin")
		received := <-rec.Received
		assert.True(t, received, "expected received receipt")
		headers := StompHeaders{}
		headers["transaction"] = "transid"
		err = client.Publish("/test/trans", "application/json", []byte(`{"test":"test"}`), headers, nil)
		assert.NoError(t, err, "did not expect an error transaction Commit")
		rec = NewReceipt(time.Second * 1)
		err = client.Commit("transid", StompHeaders{}, rec)
		assert.NoError(t,err,"did not expect an error commiting transaction")
	}
}

func TestClient_Transaction_Abort(t *testing.T) {
	if "" != SKIP_INTEGRATION || "" == INTEGRATION_SERVER {
		t.Skip("INTEGRATION DISABLED")
	}
	var tableOpts = []ClientOpts{
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.1"),
		GenerateClientOpts(INTEGRATION_SERVER, "admin", "admin", "1.2"),
	}
	for _, opts := range tableOpts {
		fmt.Println("testing version ", opts.Version)
		client := NewClient(opts)
		err := client.Connect()
		assert.NoError(t, err, "did not expect error connecting")
		rec := NewReceipt(time.Second * 1)
		_, err = client.Subscribe("/test/trans2", func(f Frame) {
			fmt.Println(f.Headers)
		}, StompHeaders{}, nil)

		err = client.Begin("transid2", StompHeaders{}, rec)
		assert.NoError(t, err, "did not expect an error transaction Begin")
		received := <-rec.Received
		assert.True(t, received, "expected received receipt")
		headers := StompHeaders{}
		headers["transaction"] = "transid2"
		err = client.Publish("/test/trans2", "application/json", []byte(`{"test":"test"}`), headers, nil)
		assert.NoError(t, err, "did not expect an error transaction Commit")
		rec = NewReceipt(time.Second * 1)
		err = client.Abort("transid", StompHeaders{}, rec)
		received = <-rec.Received
		assert.True(t,received, "expected receipt ")
		assert.NoError(t,err,"did not expect an error commiting transaction")
	}

}
