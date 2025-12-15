package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"rickshaw-app/internal/config"
	"rickshaw-app/internal/middleware"
	"rickshaw-app/internal/models"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db  *gorm.DB
	rdb *redis.Client
	cfg *config.Config
}

func NewAuthHandler(db *gorm.DB, rdb *redis.Client, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, rdb: rdb, cfg: cfg}
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	UserType string `json:"user_type"`
}

type LoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.UserType != "rider" && req.UserType != "driver" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "user_type must be rider or driver"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	user := &models.User{
		Name:     req.Name,
		Phone:    req.Phone,
		Password: string(hashedPassword),
		UserType: req.UserType,
	}

	if err := h.db.Create(user).Error; err != nil {
		respondJSON(w, http.StatusConflict, map[string]string{"error": "phone already exists"})
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.UserType, h.cfg.JWTSecret)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	ctx := context.Background()
	h.rdb.Set(ctx, "token:"+user.ID, token, 24*time.Hour)

	respondJSON(w, http.StatusCreated, AuthResponse{Token: token, User: user})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	var user models.User
	if err := h.db.Where("phone = ?", req.Phone).First(&user).Error; err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.UserType, h.cfg.JWTSecret)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	ctx := context.Background()
	h.rdb.Set(ctx, "token:"+user.ID, token, 24*time.Hour)

	respondJSON(w, http.StatusOK, AuthResponse{Token: token, User: &user})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	ctx := context.Background()
	h.rdb.Del(ctx, "token:"+userID)

	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
