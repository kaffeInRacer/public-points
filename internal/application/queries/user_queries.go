package queries

import (
	"online-shop/internal/domain/user"
)

type GetUserProfileQuery struct {
	UserID string `json:"user_id" validate:"required"`
}

type ListUsersQuery struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type GetUserProfileQueryHandler struct {
	userRepo user.Repository
}

func NewGetUserProfileQueryHandler(userRepo user.Repository) *GetUserProfileQueryHandler {
	return &GetUserProfileQueryHandler{userRepo: userRepo}
}

func (h *GetUserProfileQueryHandler) Handle(query GetUserProfileQuery) (*user.User, error) {
	return h.userRepo.GetByID(query.UserID)
}

type ListUsersQueryHandler struct {
	userRepo user.Repository
}

func NewListUsersQueryHandler(userRepo user.Repository) *ListUsersQueryHandler {
	return &ListUsersQueryHandler{userRepo: userRepo}
}

func (h *ListUsersQueryHandler) Handle(query ListUsersQuery) ([]*user.User, error) {
	if query.Limit <= 0 {
		query.Limit = 10
	}
	return h.userRepo.List(query.Limit, query.Offset)
}