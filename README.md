# Stompy A golang stomp client. Supports stomp vers 1.1 and 1.2


Currently a work in progress. Stomp client for golang

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
        err = client.Publish("/test/test", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, nil)
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
        err = client.Publish("/test/test", "application/json", []byte(`{"test":"test"}`), StompHeaders{}, rec)
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


