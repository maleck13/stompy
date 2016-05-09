# Stompy A golang stomp client. Supports stomp vers 1.1 and 1.2


Golang Stomp client.

## Installing

```bash

    go get github.com/maleck13/stompy
  
```
  
# Usage
  
#### Example Publish
  
  ```golang
        opts := stompy.ClientOpts{
                		HostAndPort: "localhost:61613",
                		Timeout:     20 * time.Second,
                		Vhost:       "localhost",
                		User:        "user",
                		PassCode:    "pass",
                		Version:     "1.2",
                	}
        client := NewClient(opts)
        err := client.Connect()
        if err != nil{
          fmt.Fatal(err)
        }
        sendHeaders := stompy.StompHeaders{}
        sendHeaders["content-type"] = "application/json"
        err = client.Publish("/test/test", []byte(`{"test":"test"}`), sendHeaders, nil)
        if err != nil{
            fmt.Fatal(err)
        }
        
        
```     


#### Example Publish with receipt
  
  ```golang
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
        sendHeaders := stompy.StompHeaders{}
        sendHeaders["content-type"] = "application/json"
        err = client.Publish("/test/test", "application/json", []byte(`{"test":"test"}`), sendHeaders, rec)
        if err != nil{
            fmt.Fatal(err)
        }
        //block until the receipt is received or the timeout fires
        received := <- rec.Receipt
        
        
        
```     


#### Example Subscribe and Unsubscribe

  ```golang
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
        subId, err = client.Subscribe("/test/test", func(f Frame) {
			      fmt.Println(f.Headers, string(f.Body))
		    }, StompHeaders{}, nil)
        
        //to unsubscribe
        err = client.Unsubscribe(subId, StompHeaders{}, nil)
        
  ```      


