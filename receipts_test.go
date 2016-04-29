package stompy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReceipts_Add_Timeout(t *testing.T) {
	receipt := NewReceipt(time.Millisecond * 100)
	err := awaitingReceipt.Add("testid", receipt)
	assert.NoError(t, err, "did not expect an error adding receipt")
	received := <-receipt.Received
	assert.False(t, received, "expected received to false")
	receipt = awaitingReceipt.Get("testid")
	assert.Nil(t, receipt, "receipt should be cleaned up")
}

func TestReceipts_Add_Duplicate(t *testing.T) {
	receipt := NewReceipt(time.Millisecond * 100)
	err := awaitingReceipt.Add("testid", receipt)
	assert.NoError(t, err, "did not expect an error adding receipt")
	err = awaitingReceipt.Add("testid", receipt)
	assert.Error(t, err, "expected an error adding duplicate receipt")
}

func TestReceipts_Add(t *testing.T) {
	receipt := NewReceipt(time.Millisecond * 100)
	err := awaitingReceipt.Add("test2", receipt)
	assert.NoError(t, err, "did not expect an error adding receipt")
	go func() {
		rec := awaitingReceipt.Get("test2")
		rec.receiptReceived <- true
	}()
	received := <-receipt.Received
	assert.True(t, received, "expected received to be true")
	receipt = awaitingReceipt.Get("test2")
	assert.Nil(t, receipt, "receipt should be cleaned up")
}

func TestReceipts_Remove(t *testing.T) {
	receipt := NewReceipt(time.Millisecond * 100)
	err := awaitingReceipt.Add("test2", receipt)
	assert.NoError(t, err, "did not expect an error adding receipt")
	awaitingReceipt.Remove("test2")
	rec := awaitingReceipt.Get("test2")
	assert.Nil(t, rec, "receipt should be removed")
}
