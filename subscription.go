package stompy

import (
	"fmt"
	"sync"

	"github.com/maleck13/stompy/Godeps/_workspace/src/github.com/nu7hatch/gouuid"
)

//the subscription handler type defines the function signature that should be passed when subscribing to queues
type SubscriptionHandler func(Frame)

type subscription struct {
	Id           string
	Destination  string
	Handler      SubscriptionHandler
	AddedHeaders StompHeaders
}

func NewSubscription(destination string, handler SubscriptionHandler, headers StompHeaders) (subscription, error) {
	sub := subscription{}
	id, err := uuid.NewV4()
	if nil != err {
		return sub, err
	}
	sub.Id = id.String()
	sub.Destination = destination
	sub.Handler = handler
	sub.AddedHeaders = headers

	return sub, nil
}

//lockable struct for mapping subscription ids to their handlers
type subscriptions struct {
	sync.Mutex
	subs map[string]subscription
}

func (s *subscriptions) dispatch(incoming chan Frame) {

	for f := range incoming {
		cmd := f.CommandString()
		fmt.Println("received msg in dispatch ", cmd, f)
		switch cmd {
		case "MESSAGE":
			id := f.Headers["subscription"]
			if "" == id {
				//err
			}
			fmt.Println("sub id ", id, " subs ", s.subs)
			if sub, ok := s.subs[id]; ok {
				go sub.Handler(f)
			}
			break
		case "ERROR":
			break
		case "RECIEPT":
			break

		}

	}
}

func (s *subscriptions) addSubscription(sub subscription) error {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.subs[sub.Id]; ok {
		return ClientError("subscription already exists with that id")
	}
	s.subs[sub.Id] = sub
	return nil
}
