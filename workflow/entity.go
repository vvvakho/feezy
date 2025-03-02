package workflow

import (
	"github.com/vvvakho/feezy/domain"
)

type AddItemSignal struct {
	LineItem domain.Item
}

type RemoveItemSignal struct {
	LineItem domain.Item
}

type CloseBillSignal struct {
	Route     string
	RequestID string
}

type SignalRoute struct {
	Name string
}

var AddLineItemRoute = SignalRoute{
	Name: "addLineItem",
}

var RemoveLineItemRoute = SignalRoute{
	Name: "removeLineItem",
}

var CloseBillRoute = SignalRoute{
	Name: "CloseBillSignal",
}
