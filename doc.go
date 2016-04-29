package stompy

//Stompy is a stomp client that supports stomp 1.1 and 1.2. It exposes the StompClient interface to allow simple testing

//Examples:

//Connect and publish waiting for the receipt to come back from the server. Note example shows connection and disconnection for completeness
// but normally the connection would be long lived.


/*


 opts := stompy.ClientOpts{
                		HostAndPort: "localhost:61613",
                		Timeout:     20 * time.Second,
                		Vhost:       "localhost",
                		User:        "user",
                		PassCode:    "pass",
                		Version:     "1.2",
                	}
        client := stompy.NewClient(opts)
        err := client.Connect()
        if err != nil{
          fmt.Fatal(err)
        }
        rec := stompy.NewReceipt(time.Second * 1) //timeout after waiting for longer than a second for a receipt
        err = client.Publish("/test/test", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, rec)
        if err != nil{
            fmt.Fatal(err)
        }
        //block until the receipt is received or the timeout fires
        received := <- rec.Receipt
        client.Disconnect()
 */
