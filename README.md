# Stompy


Currently a work in progress. Stomp client for golang

## Installing

```bash

    go get github.com/maleck13/stompy
  
```
  
# Usage
  
  ## Publish
  
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
  
