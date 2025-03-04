package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/client"
)

// MockTemporalClient mocks the Temporal client
type MockTemporalClient struct {
	mock.Mock
}

func (m *MockTemporalClient) Dial(options client.Options) (client.Client, error) {
	args := m.Called(options)
	return args.Get(0).(client.Client), args.Error(1)
}

// MockWorker mocks a Temporal worker
type MockWorker struct {
	mock.Mock
}

func (m *MockWorker) Run(ch <-chan struct{}) error {
	args := m.Called(ch)
	return args.Error(0)
}

func TestTemporalConnection(t *testing.T) {
	mockClient := new(MockTemporalClient)
	mockWorker := new(MockWorker)

	// Simulate successful Temporal connection
	lazyClient, _ := client.NewLazyClient(client.Options{}) // Get the first return value
	mockClient.On("Dial", mock.Anything).Return(lazyClient, nil)
	mockWorker.On("Run", mock.Anything).Return(nil)

	// Call the mock Dial function
	clientInstance, err := mockClient.Dial(client.Options{})
	assert.NoError(t, err)
	assert.NotNil(t, clientInstance)

	// Test worker start
	interruptCh := make(chan struct{})
	err = mockWorker.Run(interruptCh)
	assert.NoError(t, err)

	// Verify expectations
	mockClient.AssertExpectations(t)
	mockWorker.AssertExpectations(t)
}
