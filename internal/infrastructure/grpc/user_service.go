package grpc

import (
	"context"
	"fmt"
	"time"


	"online-shop/internal/domain/user"
	"online-shop/internal/infrastructure/redis"
	"online-shop/internal/infrastructure/database"
	pb "online-shop/proto/generated/online-shop/proto/user"
	"online-shop/pkg/jwt"
	
	"go.uber.org/zap"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	userRepo    *database.UserRepository
	cacheClient *redis.RedisClient
	jwtService  *jwt.JWTManager
	logger      *zap.Logger
}

func NewUserServiceServer(
	userRepo *database.UserRepository,
	cacheClient *redis.RedisClient,
	jwtService *jwt.JWTManager,
	logger *zap.Logger,
) *UserServiceServer {
	return &UserServiceServer{
		userRepo:    userRepo,
		cacheClient: cacheClient,
		jwtService:  jwtService,
		logger:      logger,
	}
}

func (s *UserServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.logger.Info("User registration request", zap.String("email", req.Email))

	// Validate input
	if req.Email == "" || req.Password == "" || req.FirstName == "" {
		return &pb.RegisterResponse{
			Success: false,
			Message: "Email, password, and first name are required",
		}, nil
	}

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(req.Email)
	if err == nil && existingUser != nil {
		return &pb.RegisterResponse{
			Success: false,
			Message: "User with this email already exists",
		}, nil
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to process password")
	}

	// Create user entity
	user := &user.User{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Role:      "customer",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save user to database
	if err := s.userRepo.Create(user); err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to create user")
	}

	// Generate tokens
	accessToken, err := s.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		s.logger.Error("Failed to generate access token", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to generate access token")
	}

	refreshToken, err := s.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		s.logger.Error("Failed to generate refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to generate refresh token")
	}

	// Cache user data
	userKey := fmt.Sprintf("user:%s", user.ID)
	if err := s.cacheClient.Set(userKey, user, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to cache user data", zap.Error(err))
	}

	// Cache refresh token
	refreshKey := fmt.Sprintf("refresh_token:%s", user.ID)
	if err := s.cacheClient.Set(refreshKey, refreshToken, 7*24*time.Hour); err != nil {
		s.logger.Warn("Failed to cache refresh token", zap.Error(err))
	}

	s.logger.Info("User registered successfully", zap.String("user_id", user.ID), zap.String("email", user.Email))

	return &pb.RegisterResponse{
		Success:      true,
		Message:      "User registered successfully",
		User:         s.entityToProto(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *UserServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	s.logger.Info("User login request", zap.String("email", req.Email))

	// Validate input
	if req.Email == "" || req.Password == "" {
		return &pb.LoginResponse{
			Success: false,
			Message: "Email and password are required",
		}, nil
	}

	// Get user from database
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil || user == nil {
		return &pb.LoginResponse{
			Success: false,
			Message: "Invalid email or password",
		}, nil
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return &pb.LoginResponse{
			Success: false,
			Message: "Invalid email or password",
		}, nil
	}

	// Check user status
	if user.Status != "active" {
		return &pb.LoginResponse{
			Success: false,
			Message: "Account is not active",
		}, nil
	}

	// Generate tokens
	accessToken, err := s.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		s.logger.Error("Failed to generate access token", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to generate access token")
	}

	refreshToken, err := s.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		s.logger.Error("Failed to generate refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to generate refresh token")
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Cache user data
	userKey := fmt.Sprintf("user:%s", user.ID)
	if err := s.cacheClient.Set(userKey, user, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to cache user data", zap.Error(err))
	}

	// Cache session
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	sessionData := map[string]interface{}{
		"user_id":    user.ID,
		"email":      user.Email,
		"role":       user.Role,
		"created_at": time.Now(),
	}
	if err := s.cacheClient.Set(sessionKey, sessionData, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to cache session", zap.Error(err))
	}

	// Cache refresh token
	refreshKey := fmt.Sprintf("refresh_token:%s", user.ID)
	if err := s.cacheClient.Set(refreshKey, refreshToken, 7*24*time.Hour); err != nil {
		s.logger.Warn("Failed to cache refresh token", zap.Error(err))
	}

	s.logger.Info("User logged in successfully", zap.String("user_id", user.ID), zap.String("email", user.Email))

	return &pb.LoginResponse{
		Success:      true,
		Message:      "Login successful",
		User:         s.entityToProto(user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		SessionId:    sessionID,
	}, nil
}

func (s *UserServiceServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	s.logger.Info("Refresh token request")

	// Validate refresh token
	claims, err := s.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		return &pb.RefreshTokenResponse{
			Success: false,
			Message: "Invalid refresh token",
		}, nil
	}

	userID := claims.UserID

	// Check if refresh token exists in cache
	refreshKey := fmt.Sprintf("refresh_token:%s", userID)
	cachedToken, err := s.cacheClient.GetString(refreshKey)
	if err != nil || cachedToken != req.RefreshToken {
		return &pb.RefreshTokenResponse{
			Success: false,
			Message: "Refresh token not found or expired",
		}, nil
	}

	// Get user from database
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return &pb.RefreshTokenResponse{
			Success: false,
			Message: "User not found",
		}, nil
	}

	// Generate new tokens
	newAccessToken, err := s.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		s.logger.Error("Failed to generate new access token", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to generate access token")
	}

	newRefreshToken, err := s.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		s.logger.Error("Failed to generate new refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to generate refresh token")
	}

	// Update refresh token in cache
	if err := s.cacheClient.Set(refreshKey, newRefreshToken, 7*24*time.Hour); err != nil {
		s.logger.Warn("Failed to update refresh token in cache", zap.Error(err))
	}

	s.logger.Info("Token refreshed successfully", zap.String("user_id", userID))

	return &pb.RefreshTokenResponse{
		Success:      true,
		Message:      "Token refreshed successfully",
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *UserServiceServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	s.logger.Info("Get profile request", zap.String("user_id", req.UserId))

	// Try to get user from cache first
	userKey := fmt.Sprintf("user:%s", req.UserId)
	var user *user.User
	
	if err := s.cacheClient.Get(userKey, &user); err != nil {
		// Cache miss, get from database
		var err error
		user, err = s.userRepo.GetByID(req.UserId)
		if err != nil || user == nil {
			return nil, status.Error(codes.NotFound, "User not found")
		}

		// Cache the user data
		if err := s.cacheClient.Set(userKey, user, 24*time.Hour); err != nil {
			s.logger.Warn("Failed to cache user data", zap.Error(err))
		}
	}

	return &pb.GetProfileResponse{
		User: s.entityToProto(user),
	}, nil
}

func (s *UserServiceServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	s.logger.Info("Update profile request", zap.String("user_id", req.UserId))

	// Get user from database
	user, err := s.userRepo.GetByID(req.UserId)
	if err != nil || user == nil {
		return nil, status.Error(codes.NotFound, "User not found")
	}

	// Update user fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	user.UpdatedAt = time.Now()

	// Save to database
	if err := s.userRepo.Update(user); err != nil {
		s.logger.Error("Failed to update user", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to update user")
	}

	// Update cache
	userKey := fmt.Sprintf("user:%s", user.ID)
	if err := s.cacheClient.Set(userKey, user, 24*time.Hour); err != nil {
		s.logger.Warn("Failed to update user cache", zap.Error(err))
	}

	s.logger.Info("Profile updated successfully", zap.String("user_id", user.ID))

	return &pb.UpdateProfileResponse{
		User: s.entityToProto(user),
	}, nil
}

func (s *UserServiceServer) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	s.logger.Info("Change password request", zap.String("user_id", req.UserId))

	// Get user from database
	user, err := s.userRepo.GetByID(req.UserId)
	if err != nil || user == nil {
		return &pb.ChangePasswordResponse{
			Success: false,
		}, status.Error(codes.NotFound, "User not found")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return &pb.ChangePasswordResponse{
			Success: false,
		}, status.Error(codes.InvalidArgument, "Invalid old password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash new password", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to process password")
	}

	// Update password
	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(user); err != nil {
		s.logger.Error("Failed to update password", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to update password")
	}

	// Invalidate all sessions and refresh tokens for this user
	refreshKey := fmt.Sprintf("refresh_token:%s", user.ID)
	s.cacheClient.Delete(refreshKey)

	s.logger.Info("Password changed successfully", zap.String("user_id", user.ID))

	return &pb.ChangePasswordResponse{
		Success: true,
	}, nil
}

func (s *UserServiceServer) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	s.logger.Info("Logout request", zap.String("user_id", req.UserId), zap.String("session_id", req.SessionId))

	// Delete session from cache
	sessionKey := fmt.Sprintf("session:%s", req.SessionId)
	if err := s.cacheClient.Delete(sessionKey); err != nil {
		s.logger.Warn("Failed to delete session from cache", zap.Error(err))
	}

	// Delete refresh token from cache
	refreshKey := fmt.Sprintf("refresh_token:%s", req.UserId)
	if err := s.cacheClient.Delete(refreshKey); err != nil {
		s.logger.Warn("Failed to delete refresh token from cache", zap.Error(err))
	}

	s.logger.Info("User logged out successfully", zap.String("user_id", req.UserId))

	return &pb.LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	}, nil
}

func (s *UserServiceServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	claims, err := s.jwtService.ValidateToken(req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: "Invalid token",
		}, nil
	}

	userID := claims.UserID
	role := claims.Role

	return &pb.ValidateTokenResponse{
		Valid:   true,
		UserId:  userID,
		Role:    role,
		Message: "Token is valid",
	}, nil
}

func (s *UserServiceServer) entityToProto(user *user.User) *pb.User {
	return &pb.User{
		Id:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
		Role:      string(user.Role),
		Status:    string(user.Status),
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}