package stompy

import (
	"sync"
	"time"
)

type Receipt struct {
	Received        chan bool
	receiptReceived chan bool
	Timeout         time.Duration
}

func (rec *Receipt) cleanUp(id string) {
	awaitingReceipt.Remove(id)
	close(rec.receiptReceived)
	close(rec.Received)
}

func NewReceipt(timeout time.Duration) *Receipt {
	return &Receipt{make(chan bool, 1), make(chan bool, 1), timeout}
}

type receipts struct {
	sync.RWMutex
	receipts map[string]*Receipt
}

func (r *receipts) Add(id string, rec *Receipt) error {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.receipts[id]; ok {
		return ClientError("already a receipt with that id " + id)
	}
	r.receipts[id] = rec
	//make sure we clean up. if we receive our receipt forward it on to the client
	//after the timeout send received as false.
	go func(receipt *Receipt, id string) {
		defer receipt.cleanUp(id)

		select {
		case <-rec.receiptReceived:
			rec.Received <- true
			break
		case <-time.After(rec.Timeout):
			rec.Received <- false
			break
		}

	}(rec, id)
	return nil
}

func (r *receipts) Remove(id string) {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.receipts[id]; ok {
		delete(r.receipts, id)
		return
	}
	return
}

func (r *receipts) Count() int {
	r.RLock()
	defer r.RUnlock()
	return len(r.receipts)
}

func (r *receipts) Get(id string) *Receipt {
	r.RLock()
	defer r.RUnlock()
	return r.receipts[id]
}

var awaitingReceipt = &receipts{receipts: make(map[string]*Receipt)}
