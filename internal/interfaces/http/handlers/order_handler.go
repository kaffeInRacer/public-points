package handlers

import (
	"net/http"
	"online-shop/internal/application/commands"
	"online-shop/internal/application/queries"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	createOrderHandler    *commands.CreateOrderCommandHandler
	cancelOrderHandler    *commands.CancelOrderCommandHandler
	getOrderHandler       *queries.GetOrderQueryHandler
	getUserOrdersHandler  *queries.GetUserOrdersQueryHandler
}

func NewOrderHandler(
	createOrderHandler *commands.CreateOrderCommandHandler,
	cancelOrderHandler *commands.CancelOrderCommandHandler,
	getOrderHandler *queries.GetOrderQueryHandler,
	getUserOrdersHandler *queries.GetUserOrdersQueryHandler,
) *OrderHandler {
	return &OrderHandler{
		createOrderHandler:   createOrderHandler,
		cancelOrderHandler:   cancelOrderHandler,
		getOrderHandler:      getOrderHandler,
		getUserOrdersHandler: getUserOrdersHandler,
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var cmd commands.CreateOrderCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd.UserID = userID.(string)

	order, err := h.createOrderHandler.Handle(cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"order": order})
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	query := queries.GetOrderQuery{OrderID: orderID}
	order, err := h.getOrderHandler.Handle(query)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Check if user owns the order or is admin
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	
	if order.UserID != userID.(string) && userRole.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	query := queries.GetUserOrdersQuery{
		UserID: userID.(string),
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			query.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			query.Offset = o
		}
	}

	orders, err := h.getUserOrdersHandler.Handle(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	cmd := commands.CancelOrderCommand{
		OrderID: orderID,
		UserID:  userID.(string),
	}

	if err := h.cancelOrderHandler.Handle(cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled successfully"})
}

// GetCart handles getting user's cart
func (h *OrderHandler) GetCart(c *gin.Context) {
	// TODO: Implement get cart logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get cart not implemented yet"})
}

// AddToCart handles adding item to cart
func (h *OrderHandler) AddToCart(c *gin.Context) {
	// TODO: Implement add to cart logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Add to cart not implemented yet"})
}

// UpdateCartItem handles updating cart item
func (h *OrderHandler) UpdateCartItem(c *gin.Context) {
	// TODO: Implement update cart item logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Update cart item not implemented yet"})
}

// RemoveFromCart handles removing item from cart
func (h *OrderHandler) RemoveFromCart(c *gin.Context) {
	// TODO: Implement remove from cart logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Remove from cart not implemented yet"})
}

// ClearCart handles clearing the cart
func (h *OrderHandler) ClearCart(c *gin.Context) {
	// TODO: Implement clear cart logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Clear cart not implemented yet"})
}

// ProcessPayment handles payment processing
func (h *OrderHandler) ProcessPayment(c *gin.Context) {
	// TODO: Implement payment processing logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Payment processing not implemented yet"})
}

// GetInvoice handles getting order invoice
func (h *OrderHandler) GetInvoice(c *gin.Context) {
	// TODO: Implement get invoice logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get invoice not implemented yet"})
}

// GetOrders handles getting all orders (admin)
func (h *OrderHandler) GetOrders(c *gin.Context) {
	// TODO: Implement get all orders logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Get orders not implemented yet"})
}

// UpdateOrderStatus handles updating order status (admin)
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	// TODO: Implement update order status logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Update order status not implemented yet"})
}

// ShipOrder handles shipping an order (admin)
func (h *OrderHandler) ShipOrder(c *gin.Context) {
	// TODO: Implement ship order logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Ship order not implemented yet"})
}

// RefundOrder handles refunding an order (admin)
func (h *OrderHandler) RefundOrder(c *gin.Context) {
	// TODO: Implement refund order logic
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Refund order not implemented yet"})
}