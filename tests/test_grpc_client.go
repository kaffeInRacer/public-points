package main

import (
	"context"
	"fmt"
	"log"
	"time"

	userPb "online-shop/proto/generated/online-shop/proto/user"
	productPb "online-shop/proto/generated/online-shop/proto/product"
	orderPb "online-shop/proto/generated/online-shop/proto/order"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func runGrpcClient() {
	// Connect to gRPC server
	conn, err := grpc.Dial("localhost:12001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	// Create clients
	userClient := userPb.NewUserServiceClient(conn)
	productClient := productPb.NewProductServiceClient(conn)
	orderClient := orderPb.NewOrderServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Println("=== Testing gRPC Services ===")

	// Test User Service - Register
	fmt.Println("\n1. Testing User Registration...")
	registerReq := &userPb.RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Phone:     "1234567890",
	}

	registerResp, err := userClient.Register(ctx, registerReq)
	if err != nil {
		fmt.Printf("Register failed: %v\n", err)
	} else {
		fmt.Printf("Register successful: %+v\n", registerResp)
	}

	// Test User Service - Login
	fmt.Println("\n2. Testing User Login...")
	loginReq := &userPb.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	loginResp, err := userClient.Login(ctx, loginReq)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
	} else {
		fmt.Printf("Login successful: Token length = %d\n", len(loginResp.AccessToken))
	}

	// Test Product Service - Get Products
	fmt.Println("\n3. Testing Get Products...")
	productsReq := &productPb.GetProductsRequest{
		Limit: 10,
	}

	productsResp, err := productClient.GetProducts(ctx, productsReq)
	if err != nil {
		fmt.Printf("Get products failed: %v\n", err)
	} else {
		fmt.Printf("Get products successful: Found %d products\n", len(productsResp.Products))
	}

	// Test Product Service - Search Products
	fmt.Println("\n4. Testing Search Products...")
	searchReq := &productPb.SearchProductsRequest{
		Query: "test",
	}

	searchResp, err := productClient.SearchProducts(ctx, searchReq)
	if err != nil {
		fmt.Printf("Search products failed: %v\n", err)
	} else {
		fmt.Printf("Search products successful: Found %d products\n", len(searchResp.Products))
	}

	// Test Order Service - Get User Orders (if we have a token)
	if loginResp != nil && loginResp.AccessToken != "" {
		fmt.Println("\n5. Testing Get User Orders...")
		ordersReq := &orderPb.GetUserOrdersRequest{
			UserId: "1", // Assuming user ID 1
		}

		ordersResp, err := orderClient.GetUserOrders(ctx, ordersReq)
		if err != nil {
			fmt.Printf("Get user orders failed: %v\n", err)
		} else {
			fmt.Printf("Get user orders successful: Found %d orders\n", len(ordersResp.Orders))
		}
	}

	fmt.Println("\n=== gRPC Service Testing Complete ===")
}