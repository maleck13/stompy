package stompy

import (
	"time"
)

func GenerateClientOpts(host, user, pass, vers string) ClientOpts {
	opts := ClientOpts{
		HostAndPort: host,
		Timeout:     20 * time.Second,
		Vhost:       "localhost",
		User:        user,
		PassCode:    pass,
		Version:     vers,
	}
	return opts
}
