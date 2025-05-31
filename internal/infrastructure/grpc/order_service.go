package grpc

import (
	"context"
	"fmt"
	"time"

	"online-shop/internal/domain/order"
	paymentDomain "online-shop/internal/domain/payment"
	"online-shop/internal/infrastructure/redis"
	"online-shop/internal/infrastructure/database"
	"online-shop/internal/infrastructure/payment"
	pb "online-shop/online-shop/proto/order"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderServiceServer struct {
	pb.UnimplementedOrderServiceServer
	orderRepo       *database.OrderRepository
	productRepo     *database.ProductRepository
	userRepo        *database.UserRepository
	paymentRepo     *database.PaymentRepository
	cacheClient     *redis.RedisClient
	paymentProvider *payment.MidtransProvider
	logger          *zap.Logger
}

func NewOrderServiceServer(
	orderRepo *database.OrderRepository,
	productRepo *database.ProductRepository,
	userRepo *database.UserRepository,
	paymentRepo *database.PaymentRepository,
	cacheClient *redis.RedisClient,
	paymentProvider *payment.MidtransProvider,
	logger *zap.Logger,
) *OrderServiceServer {
	return &OrderServiceServer{
		orderRepo:       orderRepo,
		productRepo:     productRepo,
		userRepo:        userRepo,
		paymentRepo:     paymentRepo,
		cacheClient:     cacheClient,
		paymentProvider: paymentProvider,
		logger:          logger,
	}
}

func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	s.logger.Info("Create order request", zap.String("user_id", req.UserId), zap.Int("items_count", len(req.Items)))

	// Validate input
	if req.UserId == "" || len(req.Items) == 0 {
		return &pb.CreateOrderResponse{
			Success: false,
			Message: "User ID and items are required",
		}, nil
	}

	// Verify user exists
	user, err := s.userRepo.GetByID(req.UserId)
	if err != nil || user == nil {
		return &pb.CreateOrderResponse{
			Success: false,
			Message: "User not found",
		}, nil
	}

	// Create order entity
	orderEntity := &order.Order{
		ID:     uuid.New().String(),
		UserID: req.UserId,
		Status: order.StatusPending,
		ShippingAddress: order.Address{
			Street: req.ShippingAddress, // For now, store as street field
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var totalAmount float64
	var orderItems []*order.OrderItem

	// Process each item
	for _, item := range req.Items {
		// Get product details
		product, err := s.productRepo.GetByID(item.ProductId)
		if err != nil || product == nil {
			return &pb.CreateOrderResponse{
				Success: false,
				Message: fmt.Sprintf("Product %s not found", item.ProductId),
			}, nil
		}

		// Check stock availability
		if product.Stock < int(item.Quantity) {
			return &pb.CreateOrderResponse{
				Success: false,
				Message: fmt.Sprintf("Insufficient stock for product %s", product.Name),
			}, nil
		}

		// Create order item
		orderItem := &order.OrderItem{
			ID:        uuid.New().String(),
			OrderID:   orderEntity.ID,
			ProductID: item.ProductId,
			Price:     product.Price,
			Quantity:  int(item.Quantity),
			Subtotal:  product.Price * float64(item.Quantity),
		}

		orderItems = append(orderItems, orderItem)
		totalAmount += orderItem.Subtotal

		// Update product stock
		product.Stock -= int(item.Quantity)
		if err := s.productRepo.Update(product); err != nil {
			s.logger.Error("Failed to update product stock", zap.String("product_id", product.ID), zap.Error(err))
			return nil, status.Error(codes.Internal, "Failed to update product stock")
		}

		// Update product cache
		productKey := fmt.Sprintf("product:%s", product.ID)
		if err := s.cacheClient.Set(productKey, product, 24*time.Hour); err != nil {
			s.logger.Warn("Failed to update product cache", zap.Error(err))
		}
	}

	orderEntity.TotalAmount = totalAmount
	orderEntity.Items = make([]order.OrderItem, len(orderItems))
	for i, item := range orderItems {
		orderEntity.Items[i] = *item
	}

	// Save order to database
	if err := s.orderRepo.Create(orderEntity); err != nil {
		s.logger.Error("Failed to create order", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to create order")
	}

	// Create payment with Midtrans
	var paymentURL string
	if req.PaymentMethod != "cash_on_delivery" {
		// Create payment entity
		paymentEntity := paymentDomain.NewPayment(
			orderEntity.ID,
			req.UserId,
			totalAmount,
			paymentDomain.Method(req.PaymentMethod),
		)

		// Create payment with provider
		paymentResp, err := s.paymentProvider.CreatePayment(paymentEntity)
		if err != nil {
			s.logger.Error("Failed to create payment", zap.Error(err))
			// Don't fail the order creation, just log the error
		} else {
			paymentURL = paymentResp.PaymentURL
			
			// Update payment with provider response
			paymentEntity.ExternalID = paymentResp.ExternalID
			paymentEntity.PaymentURL = paymentResp.PaymentURL
			paymentEntity.TransactionID = paymentResp.TransactionID
			paymentEntity.ExpiresAt = paymentResp.ExpiresAt

			// Save payment to database
			if err := s.paymentRepo.Create(paymentEntity); err != nil {
				s.logger.Error("Failed to save payment", zap.Error(err))
			} else {
				// Update order with payment ID
				orderEntity.PaymentID = paymentEntity.ID
				s.orderRepo.Update(orderEntity)
			}
		}
	}

	// Cache order
	orderKey := fmt.Sprintf("order:%s", orderEntity.ID)
	if err := s.cacheClient.Set(orderKey, orderEntity, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to cache order", zap.Error(err))
	}

	// Invalidate user orders cache
	s.invalidateUserOrdersCache(req.UserId)

	s.logger.Info("Order created successfully", zap.String("order_id", orderEntity.ID), zap.String("user_id", req.UserId), zap.Float64("total_amount", totalAmount))

	return &pb.CreateOrderResponse{
		Success:    true,
		Message:    "Order created successfully",
		Order:      s.entityToProto(orderEntity),
		PaymentUrl: paymentURL,
	}, nil
}

func (s *OrderServiceServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	s.logger.Info("Get order request", zap.String("order_id", req.OrderId), zap.String("user_id", req.UserId))

	// Try to get from cache first
	orderKey := fmt.Sprintf("order:%s", req.OrderId)
	var orderEntity *order.Order
	
	if err := s.cacheClient.Get(orderKey, &orderEntity); err != nil {
		// Cache miss, get from database
		var err error
		orderEntity, err = s.orderRepo.GetByID(req.OrderId)
		if err != nil || orderEntity == nil {
			return &pb.GetOrderResponse{
				Success: false,
				Message: "Order not found",
			}, nil
		}

		// Cache the order
		if err := s.cacheClient.Set(orderKey, orderEntity, 24*time.Hour); err != nil {
			s.logger.Warn("Failed to cache order", zap.Error(err))
		}
	}

	// Check if user has access to this order
	if orderEntity.UserID != req.UserId {
		return &pb.GetOrderResponse{
			Success: false,
			Message: "Access denied",
		}, nil
	}

	return &pb.GetOrderResponse{
		Success: true,
		Message: "Order retrieved successfully",
		Order:   s.entityToProto(orderEntity),
	}, nil
}

func (s *OrderServiceServer) GetUserOrders(ctx context.Context, req *pb.GetUserOrdersRequest) (*pb.GetUserOrdersResponse, error) {
	s.logger.Info("Get user orders request", zap.String("user_id", req.UserId))

	// Set default values
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := int(req.Offset)
	if offset < 0 {
		offset = 0
	}

	// Try cache first
	cacheKey := fmt.Sprintf("user_orders:%s:%d:%d:%s", req.UserId, limit, offset, req.Status)
	var cachedResult struct {
		Orders []*order.Order `json:"orders"`
		Total  int64           `json:"total"`
	}

	if err := s.cacheClient.Get(cacheKey, &cachedResult); err == nil {
		// Cache hit
		protoOrders := make([]*pb.Order, len(cachedResult.Orders))
		for i, order := range cachedResult.Orders {
			protoOrders[i] = s.entityToProto(order)
		}

		return &pb.GetUserOrdersResponse{
			Success: true,
			Message: "Orders retrieved successfully",
			Orders:  protoOrders,
			Total:   cachedResult.Total,
		}, nil
	}

	// Get from database
	orders, err := s.orderRepo.GetByUserID(req.UserId, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get user orders", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to get user orders")
	}

	// Filter by status if provided
	var filteredOrders []*order.Order
	if req.Status != "" {
		for _, o := range orders {
			if string(o.Status) == req.Status {
				filteredOrders = append(filteredOrders, o)
			}
		}
	} else {
		filteredOrders = orders
	}

	total := int64(len(filteredOrders))

	// Cache the result
	cachedResult.Orders = filteredOrders
	cachedResult.Total = total
	if err := s.cacheClient.Set(cacheKey, cachedResult, 10*time.Minute); err != nil {
		s.logger.Warn("Failed to cache user orders", zap.Error(err))
	}

	// Convert to proto
	protoOrders := make([]*pb.Order, len(filteredOrders))
	for i, order := range filteredOrders {
		protoOrders[i] = s.entityToProto(order)
	}

	return &pb.GetUserOrdersResponse{
		Success: true,
		Message: "Orders retrieved successfully",
		Orders:  protoOrders,
		Total:   total,
	}, nil
}

func (s *OrderServiceServer) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.UpdateOrderStatusResponse, error) {
	s.logger.Info("Update order status request", zap.String("order_id", req.OrderId), zap.String("status", req.Status), zap.String("updated_by", req.UpdatedBy))

	// Get order from database
	orderEntity, err := s.orderRepo.GetByID(req.OrderId)
	if err != nil || orderEntity == nil {
		return &pb.UpdateOrderStatusResponse{
			Success: false,
			Message: "Order not found",
		}, nil
	}

	// Validate status transition
	validStatuses := []string{"pending", "confirmed", "processing", "shipped", "delivered", "cancelled"}
	isValidStatus := false
	for _, status := range validStatuses {
		if status == req.Status {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		return &pb.UpdateOrderStatusResponse{
			Success: false,
			Message: "Invalid order status",
		}, nil
	}

	// Update order status
	orderEntity.Status = order.Status(req.Status)
	orderEntity.UpdatedAt = time.Now()

	// Save to database
	if err := s.orderRepo.Update(orderEntity); err != nil {
		s.logger.Error("Failed to update order status", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to update order status")
	}

	// Update cache
	orderKey := fmt.Sprintf("order:%s", orderEntity.ID)
	if err := s.cacheClient.Set(orderKey, orderEntity, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to update order cache", zap.Error(err))
	}

	// Invalidate user orders cache
	s.invalidateUserOrdersCache(orderEntity.UserID)

	s.logger.Info("Order status updated successfully", zap.String("order_id", orderEntity.ID), zap.String("new_status", string(orderEntity.Status)))

	return &pb.UpdateOrderStatusResponse{
		Success: true,
		Message: "Order status updated successfully",
		Order:   s.entityToProto(orderEntity),
	}, nil
}

func (s *OrderServiceServer) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	s.logger.Info("Cancel order request", zap.String("order_id", req.OrderId), zap.String("user_id", req.UserId), zap.String("reason", req.Reason))

	// Get order from database
	orderEntity, err := s.orderRepo.GetByID(req.OrderId)
	if err != nil || orderEntity == nil {
		return &pb.CancelOrderResponse{
			Success: false,
			Message: "Order not found",
		}, nil
	}

	// Check if user has access to this order
	if orderEntity.UserID != req.UserId {
		return &pb.CancelOrderResponse{
			Success: false,
			Message: "Access denied",
		}, nil
	}

	// Check if order can be cancelled
	if orderEntity.Status == order.StatusDelivered || orderEntity.Status == order.StatusCancelled {
		return &pb.CancelOrderResponse{
			Success: false,
			Message: "Order cannot be cancelled",
		}, nil
	}

	// Restore product stock
	for _, item := range orderEntity.Items {
		product, err := s.productRepo.GetByID(item.ProductID)
		if err == nil && product != nil {
			product.Stock += item.Quantity
			if err := s.productRepo.Update(product); err != nil {
				s.logger.Error("Failed to restore product stock", zap.String("product_id", product.ID), zap.Error(err))
			}

			// Update product cache
			productKey := fmt.Sprintf("product:%s", product.ID)
			if err := s.cacheClient.Set(productKey, product, 24*time.Hour); err != nil {
				s.logger.Warn("Failed to update product cache", zap.Error(err))
			}
		}
	}

	// Update order status
	orderEntity.Status = order.StatusCancelled
	orderEntity.UpdatedAt = time.Now()

	// Save to database
	if err := s.orderRepo.Update(orderEntity); err != nil {
		s.logger.Error("Failed to cancel order", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to cancel order")
	}

	// Update cache
	orderKey := fmt.Sprintf("order:%s", orderEntity.ID)
	if err := s.cacheClient.Set(orderKey, orderEntity, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to update order cache", zap.Error(err))
	}

	// Invalidate user orders cache
	s.invalidateUserOrdersCache(orderEntity.UserID)

	s.logger.Info("Order cancelled successfully", zap.String("order_id", orderEntity.ID), zap.String("reason", req.Reason))

	return &pb.CancelOrderResponse{
		Success: true,
		Message: "Order cancelled successfully",
		Order:   s.entityToProto(orderEntity),
	}, nil
}

func (s *OrderServiceServer) ProcessPayment(ctx context.Context, req *pb.ProcessPaymentRequest) (*pb.ProcessPaymentResponse, error) {
	s.logger.Info("Process payment request", zap.String("order_id", req.OrderId), zap.String("payment_method", req.PaymentMethod))

	// Get order from database
	orderEntity, err := s.orderRepo.GetByID(req.OrderId)
	if err != nil || orderEntity == nil {
		return &pb.ProcessPaymentResponse{
			Success: false,
			Message: "Order not found",
		}, nil
	}

	// Get payment status from Midtrans
	paymentResp, err := s.paymentProvider.GetPaymentStatus(orderEntity.ID)
	if err != nil {
		s.logger.Error("Failed to get payment status", zap.Error(err))
		return &pb.ProcessPaymentResponse{
			Success:       false,
			Message:       "Payment status check failed",
			PaymentStatus: "failed",
		}, nil
	}

	// Update order status based on payment
	if paymentResp.Status == paymentDomain.StatusPaid {
		orderEntity.Status = order.StatusConfirmed
	}
	orderEntity.UpdatedAt = time.Now()

	// Save to database
	if err := s.orderRepo.Update(orderEntity); err != nil {
		s.logger.Error("Failed to update order payment status", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to update order")
	}

	// Update cache
	orderKey := fmt.Sprintf("order:%s", orderEntity.ID)
	if err := s.cacheClient.Set(orderKey, orderEntity, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to update order cache", zap.Error(err))
	}

	// Invalidate user orders cache
	s.invalidateUserOrdersCache(orderEntity.UserID)

	s.logger.Info("Payment processed successfully", zap.String("order_id", orderEntity.ID), zap.String("payment_status", string(paymentResp.Status)), zap.String("transaction_id", paymentResp.TransactionID))

	return &pb.ProcessPaymentResponse{
		Success:       true,
		Message:       "Payment processed successfully",
		PaymentStatus: string(paymentResp.Status),
		TransactionId: paymentResp.TransactionID,
	}, nil
}

func (s *OrderServiceServer) entityToProto(orderEntity *order.Order) *pb.Order {
	protoItems := make([]*pb.OrderItem, len(orderEntity.Items))
	for i, item := range orderEntity.Items {
		protoItems[i] = &pb.OrderItem{
			Id:          item.ID,
			ProductId:   item.ProductID,
			ProductName: "", // TODO: Get from product repository
			Price:       item.Price,
			Quantity:    int32(item.Quantity),
			Subtotal:    item.Subtotal,
		}
	}

	return &pb.Order{
		Id:              orderEntity.ID,
		UserId:          orderEntity.UserID,
		Status:          string(orderEntity.Status),
		TotalAmount:     orderEntity.TotalAmount,
		ShippingAddress: orderEntity.ShippingAddress.Street,
		Items:           protoItems,
		CreatedAt:       timestamppb.New(orderEntity.CreatedAt),
		UpdatedAt:       timestamppb.New(orderEntity.UpdatedAt),
	}
}

func (s *OrderServiceServer) invalidateUserOrdersCache(userID string) {
	pattern := fmt.Sprintf("user_orders:%s:*", userID)
	if err := s.cacheClient.DeletePattern(pattern); err != nil {
		s.logger.Warn("Failed to invalidate user orders cache", zap.Error(err))
	}
}