package storage_test

import (
	"os"
	"testing"
	"time"

	"gitlab.ozon.dev/qwestard/homework/internal/models"
	"gitlab.ozon.dev/qwestard/homework/internal/storage"

	"github.com/stretchr/testify/assert"
)

const testFile = "test_orders.json"

func setupStorage(t *testing.T) *storage.OrderStorage {
	_ = os.Remove(testFile)
	st, err := storage.New(testFile)
	assert.NoError(t, err)
	return st
}

func TestAcceptOrder(t *testing.T) {
	st := setupStorage(t)

	orderID := "order123"
	userID := "user42"
	deadline := time.Now().Add(24 * time.Hour)

	err := st.AcceptOrderFromCourier(orderID, userID, deadline)
	assert.NoError(t, err, "Прием заказа не должен выдавать ошибку")

	orders, err := st.GetOrders(userID, 0, false)
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, orderID, orders[0].ID)
	assert.Equal(t, models.OrderStateAccepted, orders[0].CurrentState())
}

func TestDeliverOrder(t *testing.T) {
	st := setupStorage(t)

	orderID := "order789"
	userID := "user88"
	deadline := time.Now().Add(24 * time.Hour)

	err := st.AcceptOrderFromCourier(orderID, userID, deadline)
	assert.NoError(t, err)

	err = st.DeliverOrReturnClientOrders(userID, []string{orderID}, "deliver")
	assert.NoError(t, err, "Выдача заказа должна проходить без ошибок")

	orders, err := st.GetOrders(userID, 0, false)
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, models.OrderStateDelivered, orders[0].CurrentState())
}

func TestClientReturn(t *testing.T) {
	st := setupStorage(t)

	orderID := "order987"
	userID := "user99"
	deadline := time.Now().Add(24 * time.Hour)

	err := st.AcceptOrderFromCourier(orderID, userID, deadline)
	assert.NoError(t, err)

	err = st.DeliverOrReturnClientOrders(userID, []string{orderID}, "deliver")
	assert.NoError(t, err)

	err = st.DeliverOrReturnClientOrders(userID, []string{orderID}, "return")
	assert.NoError(t, err, "Клиент должен иметь возможность вернуть заказ в течение 48 часов")

	orders, err := st.GetOrders(userID, 0, false)
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, models.OrderStateClientRtn, orders[0].CurrentState())
}

func TestReturnOrderToCourier_Accepted(t *testing.T) {
	st := setupStorage(t)
	orderID := "orderAccepted"
	userID := "userTest"
	deadline := time.Now().Add(24 * time.Hour)

	err := st.AcceptOrderFromCourier(orderID, userID, deadline)
	assert.NoError(t, err)

	err = st.ReturnOrderToCourier(orderID)
	assert.NoError(t, err)

	orders, err := st.GetOrders(userID, 0, false)
	assert.NoError(t, err)
	assert.Len(t, orders, 0)
}

func TestReturnOrderToCourier_ClientRtn(t *testing.T) {
	st := setupStorage(t)
	orderID := "orderClientRtn"
	userID := "userTest"
	deadline := time.Now().Add(24 * time.Hour)

	err := st.AcceptOrderFromCourier(orderID, userID, deadline)
	assert.NoError(t, err)

	err = st.DeliverOrReturnClientOrders(userID, []string{orderID}, "deliver")
	assert.NoError(t, err)
	err = st.DeliverOrReturnClientOrders(userID, []string{orderID}, "return")
	assert.NoError(t, err)
	orders, err := st.GetOrders(userID, 0, false)
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, models.OrderStateClientRtn, orders[0].CurrentState())

	err = st.ReturnOrderToCourier(orderID)
	assert.NoError(t, err)

	orders, err = st.GetOrders(userID, 0, false)
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, models.OrderStateReturned, orders[0].CurrentState())
	assert.False(t, orders[0].ReturnedAt.IsZero(), "ReturnedAt должна быть установлена")
}

func TestGetOrders(t *testing.T) {
	st := setupStorage(t)

	err := st.AcceptOrderFromCourier("order1", "user100", time.Now().Add(24*time.Hour))
	assert.NoError(t, err)
	err = st.AcceptOrderFromCourier("order2", "user100", time.Now().Add(24*time.Hour))
	assert.NoError(t, err)

	orders, err := st.GetOrders("user100", 0, false)
	assert.NoError(t, err)
	assert.Len(t, orders, 2)
}

func TestGetReturns(t *testing.T) {
	st := setupStorage(t)

	err := st.AcceptOrderFromCourier("orderA", "userA", time.Now().Add(24*time.Hour))
	assert.NoError(t, err)
	err = st.AcceptOrderFromCourier("orderB", "userA", time.Now().Add(24*time.Hour))
	assert.NoError(t, err)
	err = st.DeliverOrReturnClientOrders("userA", []string{"orderA"}, "deliver")
	assert.NoError(t, err)
	err = st.DeliverOrReturnClientOrders("userA", []string{"orderB"}, "deliver")
	assert.NoError(t, err)
	err = st.DeliverOrReturnClientOrders("userA", []string{"orderA"}, "return")
	assert.NoError(t, err)

	returns, err := st.GetReturns(1, 10)
	assert.NoError(t, err)
	assert.Len(t, returns, 1)
	assert.Equal(t, "orderA", returns[0].ID)
}

func TestOrderHistory(t *testing.T) {
	st := setupStorage(t)

	err := st.AcceptOrderFromCourier("order100", "userX", time.Now().Add(24*time.Hour))
	assert.NoError(t, err)
	err = st.AcceptOrderFromCourier("order200", "userX", time.Now().Add(24*time.Hour))
	assert.NoError(t, err)
	err = st.DeliverOrReturnClientOrders("userX", []string{"order100"}, "deliver")
	assert.NoError(t, err)

	history, err := st.GetOrderHistory()
	assert.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Equal(t, "order100", history[0].ID, "Последний изменённый заказ должен быть первым в истории")
}
