package workflows

import (
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/vvvakho/feezy/billing/service/domain"
)

func (s *UnitTestSuite) Test_AddLineItem() {
	// Initialize a new bill
	bill := &domain.Bill{
		ID:     uuid.New(),
		Status: domain.BillOpen,
		Total: domain.Money{
			Amount:   0,
			Currency: "USD",
		},
		Items: []domain.Item{},
	}

	// Mock activities
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil) // Add mock for closing

	// Register delayed callbacks to simulate signals
	s.env.RegisterDelayedCallback(func() {
		// Query the bill to ensure it's initially empty
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(0, len(queriedBill.Items))

		// Send a signal to add a line item
		item := domain.Item{
			ID:           uuid.New(),
			PricePerUnit: domain.Money{Amount: 50, Currency: "USD"},
			Quantity:     1,
		}
		s.env.SignalWorkflow(AddLineItemRoute.Name, AddItemSignal{
			LineItem: item,
		})
	}, time.Millisecond*1)

	s.env.RegisterDelayedCallback(func() {
		// Query after adding item
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))
		s.Equal(domain.MinorUnit(50), queriedBill.Total.Amount)

		// Send signal to close the bill
		s.env.SignalWorkflow(CloseBillRoute.Name, CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*10)

	// Execute the workflow
	s.env.ExecuteWorkflow(BillWorkflow, bill)

	// Ensure workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	// Verify mock calls
	s.mockActivities.AssertCalled(s.T(), "AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything)
	s.mockActivities.AssertCalled(s.T(), "AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything)
}

func (s *UnitTestSuite) Test_AddLineItemMultiple() {
	// Initialize a new bill
	bill := &domain.Bill{
		ID:     uuid.New(),
		Status: domain.BillOpen,
		Total: domain.Money{
			Amount:   0,
			Currency: "USD",
		},
		Items: []domain.Item{},
	}

	// Mock activities
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil) // Add mock for closing

	itemID := uuid.New()

	// Register delayed callbacks to simulate signals
	s.env.RegisterDelayedCallback(func() {
		// Query the bill to ensure it's initially empty
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(0, len(queriedBill.Items))

		// Send a signal to add a line item
		item := domain.Item{
			ID:           itemID,
			PricePerUnit: domain.Money{Amount: 50, Currency: "USD"},
			Quantity:     1,
		}

		s.env.SignalWorkflow(AddLineItemRoute.Name, AddItemSignal{
			LineItem: item,
		})
	}, time.Millisecond*1)

	s.env.RegisterDelayedCallback(func() {
		time.Sleep(time.Millisecond * 5)

		// Query the bill to ensure first item was added
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))

		// Send a signal to add 3 more of the same item
		item := domain.Item{
			ID:           itemID,
			PricePerUnit: domain.Money{Amount: 50, Currency: "USD"},
			Quantity:     3,
		}
		s.env.SignalWorkflow(AddLineItemRoute.Name, AddItemSignal{
			LineItem: item,
		})
	}, time.Millisecond*5)

	s.env.RegisterDelayedCallback(func() {
		time.Sleep(time.Millisecond * 10)

		// Query after adding multiple of the existing item
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))               // len() represents unique items in bill
		s.Equal(int64(4), queriedBill.Items[0].Quantity) // here we check the quantity of a specific item
		s.Equal(domain.MinorUnit(200), queriedBill.Total.Amount)

		// Send signal to close the bill
		s.env.SignalWorkflow(CloseBillRoute.Name, CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*15)

	// Execute the workflow
	s.env.ExecuteWorkflow(BillWorkflow, bill)

	// Ensure workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	// Verify mock calls
	s.mockActivities.AssertCalled(s.T(), "AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything)
	s.mockActivities.AssertCalled(s.T(), "AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything)
}

func (s *UnitTestSuite) Test_AddLineItemCurrencyDiff() {
	// Initialize a new bill
	bill := &domain.Bill{
		ID:     uuid.New(),
		Status: domain.BillOpen,
		Total: domain.Money{
			Amount:   0,
			Currency: "USD",
		},
		Items: []domain.Item{},
	}

	// Mock activities
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil) // Add mock for closing

	Item := domain.Item{
		ID: uuid.New(),
		PricePerUnit: domain.Money{
			Amount:   275,
			Currency: "GEL",
		},
		Quantity: 1,
	}

	// Register delayed callbacks to simulate signals
	s.env.RegisterDelayedCallback(func() {
		// Query the bill to ensure it's initially empty
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(0, len(queriedBill.Items))

		// Send a signal to add a line item
		s.env.SignalWorkflow(AddLineItemRoute.Name, AddItemSignal{
			LineItem: Item,
		})
	}, time.Millisecond*1)

	s.env.RegisterDelayedCallback(func() {
		// Query after adding item
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))
		// Check that total bill amount reflects converted currency of item price
		expAmount, err := domain.Convert(queriedBill.Total.Currency, Item.PricePerUnit.Currency, Item.PricePerUnit.Amount)
		s.NoError(err)
		s.Equal(expAmount, queriedBill.Total.Amount)

		// Send signal to close the bill
		s.env.SignalWorkflow(CloseBillRoute.Name, CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*10)

	// Execute the workflow
	s.env.ExecuteWorkflow(BillWorkflow, bill)

	// Ensure workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	// Verify mock calls
	s.mockActivities.AssertCalled(s.T(), "AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything)
	s.mockActivities.AssertCalled(s.T(), "AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything)
}

func (s *UnitTestSuite) Test_AddRemoveLineItemPriceChanged() {
	// Initialize a new bill
	bill := &domain.Bill{
		ID:     uuid.New(),
		Status: domain.BillOpen,
		Total: domain.Money{
			Amount:   0,
			Currency: "USD",
		},
		Items: []domain.Item{},
	}

	// Mock activities
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	itemID := uuid.New()

	// Add an item to the bill
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(AddLineItemRoute.Name, AddItemSignal{
			LineItem: domain.Item{
				ID:           itemID,
				PricePerUnit: domain.Money{Amount: 50, Currency: "USD"},
				Quantity:     1,
			},
		})
	}, time.Millisecond*1)

	// Ensure item was added
	s.env.RegisterDelayedCallback(func() {
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))
		s.Equal(domain.MinorUnit(50), queriedBill.Total.Amount)

		// Try to add the same item ID but with a different price (should fail)
		s.env.SignalWorkflow(AddLineItemRoute.Name, AddItemSignal{
			LineItem: domain.Item{
				ID:           itemID,                                    // Same ID as before
				PricePerUnit: domain.Money{Amount: 60, Currency: "USD"}, // Price changed
				Quantity:     1,
			},
		})
	}, time.Millisecond*5)

	// Ensure item was not added and error was raised
	s.env.RegisterDelayedCallback(func() {
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)

		s.Equal(1, len(queriedBill.Items))                      // Should still be only one item
		s.Equal(int64(1), queriedBill.Items[0].Quantity)        // Quantity should not have increased
		s.Equal(domain.MinorUnit(50), queriedBill.Total.Amount) // Total should not have changed

		// Try to remove the item with a different price (should fail)
		s.env.SignalWorkflow(RemoveLineItemRoute.Name, RemoveItemSignal{
			LineItem: domain.Item{
				ID:           itemID,                                    // Same ID
				PricePerUnit: domain.Money{Amount: 60, Currency: "USD"}, // Price changed
				Quantity:     1,
			},
		})
	}, time.Millisecond*10)

	// Ensure item was not removed and error was raised
	s.env.RegisterDelayedCallback(func() {
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)

		s.Equal(1, len(queriedBill.Items))                      // Item should still be in bill
		s.Equal(int64(1), queriedBill.Items[0].Quantity)        // Quantity should be unchanged
		s.Equal(domain.MinorUnit(50), queriedBill.Total.Amount) // Total should be unchanged

		// Close bill after testing
		s.env.SignalWorkflow(CloseBillRoute.Name, CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*15)

	// Execute the workflow
	s.env.ExecuteWorkflow(BillWorkflow, bill)

	// Ensure workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	// Verify mock calls
	s.mockActivities.AssertCalled(s.T(), "AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything)
	s.mockActivities.AssertCalled(s.T(), "AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything)
}

func (s *UnitTestSuite) Test_RemoveLineItem() {
	// Initialize a new bill
	bill := &domain.Bill{
		ID:     uuid.New(),
		Status: domain.BillOpen,
		Total: domain.Money{
			Amount:   100,
			Currency: "USD",
		},
		Items: []domain.Item{},
	}

	// Mock activities
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	item := domain.Item{
		ID:           uuid.New(),
		PricePerUnit: domain.Money{Amount: 50, Currency: "USD"},
		Quantity:     2,
	}

	// Step 1: Add an item to the bill
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(AddLineItemRoute.Name, AddItemSignal{
			LineItem: item,
		})
	}, time.Millisecond*1)

	// Step 2: Ensure item was added
	s.env.RegisterDelayedCallback(func() {
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))
		s.Equal(int64(2), queriedBill.Items[0].Quantity)
		s.Equal(domain.MinorUnit(100), queriedBill.Total.Amount)

		// Send signal to remove one quantity of the item
		s.env.SignalWorkflow(RemoveLineItemRoute.Name, RemoveItemSignal{
			LineItem: domain.Item{
				ID:           item.ID,
				PricePerUnit: item.PricePerUnit,
				Quantity:     1, // Remove one quantity
			},
		})
	}, time.Millisecond*5)

	// Step 3: Ensure item quantity is reduced
	s.env.RegisterDelayedCallback(func() {
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))
		s.Equal(int64(1), queriedBill.Items[0].Quantity)        // Only one left
		s.Equal(domain.MinorUnit(50), queriedBill.Total.Amount) // Total updated

		// Send signal to close the bill
		s.env.SignalWorkflow(CloseBillRoute.Name, CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*10)

	// Execute the workflow
	s.env.ExecuteWorkflow(BillWorkflow, bill)

	// Ensure workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	// Verify mock calls
	s.mockActivities.AssertCalled(s.T(), "AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything)
	s.mockActivities.AssertCalled(s.T(), "AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything)
}

func (s *UnitTestSuite) Test_RemoveLineItemComplete() {
	// Initialize a new bill
	bill := &domain.Bill{
		ID:     uuid.New(),
		Status: domain.BillOpen,
		Total: domain.Money{
			Amount:   150,
			Currency: "USD",
		},
		Items: []domain.Item{},
	}

	// Mock activities
	s.mockActivities.On("AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.mockActivities.On("AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	item := domain.Item{
		ID:           uuid.New(),
		PricePerUnit: domain.Money{Amount: 50, Currency: "USD"},
		Quantity:     3,
	}

	// Step 1: Add an item to the bill
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(AddLineItemRoute.Name, AddItemSignal{
			LineItem: item,
		})
	}, time.Millisecond*1)

	// Step 2: Ensure item was added
	s.env.RegisterDelayedCallback(func() {
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(1, len(queriedBill.Items))
		s.Equal(int64(3), queriedBill.Items[0].Quantity)
		s.Equal(domain.MinorUnit(150), queriedBill.Total.Amount)

		// Send signal to remove all quantity of the item
		s.env.SignalWorkflow(RemoveLineItemRoute.Name, RemoveItemSignal{
			LineItem: domain.Item{
				ID:           item.ID,
				PricePerUnit: item.PricePerUnit,
				Quantity:     3, // Remove all quantity
			},
		})
	}, time.Millisecond*5)

	// Step 3: Ensure item is fully removed
	s.env.RegisterDelayedCallback(func() {
		res, err := s.env.QueryWorkflow("getBill")
		s.NoError(err)
		var queriedBill domain.Bill
		err = res.Get(&queriedBill)
		s.NoError(err)
		s.Equal(0, len(queriedBill.Items))                     // No items left
		s.Equal(domain.MinorUnit(0), queriedBill.Total.Amount) // Total is zero

		// Send signal to close the bill
		s.env.SignalWorkflow(CloseBillRoute.Name, CloseBillSignal{
			Route:     "CloseBillRoute",
			RequestID: uuid.NewString(),
		})
	}, time.Millisecond*10)

	// Execute the workflow
	s.env.ExecuteWorkflow(BillWorkflow, bill)

	// Ensure workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	// Verify mock calls
	s.mockActivities.AssertCalled(s.T(), "AddOpenBillToDB", mock.Anything, mock.Anything, mock.Anything)
	s.mockActivities.AssertCalled(s.T(), "AddClosedBillToDB", mock.Anything, mock.Anything, mock.Anything)
}
