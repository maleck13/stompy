package stompy

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCantAddTwoSubscriptionsWithSameId(t *testing.T)  {

	sub,err := newSubscription("test/test",func(f Frame){},StompHeaders{})
	assert.NoError(t,err,"did not expect an error creating a subscription")
	subs := newSubscriptions()
	assert.NotNil(t,subs,"excpected subs not to be nil")
	err = subs.addSubscription(sub)
	assert.NoError(t,err, "did not expect an error adding subcription")
	err = subs.addSubscription(sub)
	assert.Error(t,err,"expected an error adding subscription second time")

}
