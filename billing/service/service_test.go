package billing

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	temporalclient "github.com/vvvakho/feezy/billing/service/temporal"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

// Mock Temporal Client
type MockTemporalClient struct {
	mock.Mock
}

func TestInitService(t *testing.T) {
	// Mock the TemporalDial function
	temporalclient.TemporalDial = func(options client.Options) (client.Client, error) {
		return &mocks.Client{}, nil
	}

	c, err := temporalclient.InitTemporalClient()
	assert.NoError(t, err)
	assert.NotNil(t, c)

	// Test error scenario
	temporalclient.TemporalDial = func(options client.Options) (client.Client, error) {
		return nil, fmt.Errorf("connection error")
	}

	c, err = temporalclient.InitTemporalClient()
	assert.Error(t, err)
	assert.Nil(t, c)
}
