package stompy

import (
	"sync"

	"github.com/nu7hatch/gouuid"
)

//the subscription handler type defines the function signature that should be passed when subscribing to queues
type SubscriptionHandler func(Frame)

type subscription struct {
	Id           string
	Destination  string
	Handler      SubscriptionHandler
	AddedHeaders StompHeaders
}

func newSubscription(destination string, handler SubscriptionHandler, headers StompHeaders) (subscription, error) {
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

func newSubscriptions() *subscriptions {
	return &subscriptions{subs: make(map[string]subscription)}
}

func (s *subscriptions) dispatch(incoming chan Frame) {

	var forward = func(f Frame) {
		id := f.Headers["subscription"]
		if sub, ok := s.subs[id]; ok {
			go sub.Handler(f)
		}
	}

	for f := range incoming {
		cmd := f.CommandString()
		switch cmd {
		case "MESSAGE":
			forward(f)
			break
		case "ERROR":
			forward(f)
			break
		case "RECEIPT":
			if receiptId, ok := f.Headers["receipt-id"]; ok {
				if receipt := awaitingReceipt.Get(receiptId); nil != receipt {
					//if some one is listening on the channel send the receipt make sure to remove receipt
					receipt.receiptReceived <- true
				}
			}
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

func (s *subscriptions) removeSubscription(subId string) {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.subs[subId]; ok {
		delete(s.subs, subId)
	}
}
