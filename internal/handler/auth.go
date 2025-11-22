package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"backend/internal/db"
	"backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Fullname string `json:"fullname" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make(map[string]string)
			for _, fe := range ve {
				out[fe.Field()] = fe.Tag()
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": out})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := model.User{Fullname: req.Fullname, Email: req.Email, Password: string(hashed)}
	if err := db.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "user": user})
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user model.User
	if err := db.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	secret := os.Getenv("JWT_SECRET")
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))

	// generate refresh token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}
	refreshTokenStr := hex.EncodeToString(b)

	// expiry from env or default 168 hours
	hrs := 168
	if v := os.Getenv("REFRESH_TOKEN_EXP_HOURS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			hrs = parsed
		}
	}
	expiresAt := time.Now().Add(time.Duration(hrs) * time.Hour)

	rt := model.RefreshToken{UserID: user.ID, Token: refreshTokenStr, ExpiresAt: expiresAt}
	if err := db.DB.Create(&rt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "access_token": tokenStr, "token_type": "Bearer", "refresh_token": refreshTokenStr, "user": user})
}

func Me(c *gin.Context) {
	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var user model.User
	if err := db.DB.First(&user, uid.(uint)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func Logout(c *gin.Context) {
	// accept refresh_token in body to revoke it
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&body); err == nil && body.RefreshToken != "" {
		db.DB.Where("token = ?", body.RefreshToken).Delete(&model.RefreshToken{})
		c.JSON(http.StatusOK, gin.H{"message": "Refresh token revoked"})
		return
	}

	// fallback: if auth middleware set user_id, delete all refresh tokens for user
	if uid, ok := c.Get("user_id"); ok {
		db.DB.Where("user_id = ?", uid.(uint)).Delete(&model.RefreshToken{})
		c.JSON(http.StatusOK, gin.H{"message": "All refresh tokens revoked for user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "No token provided"})
}

// Refresh exchanges a valid refresh token for a new access token and rotates refresh token
func Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token required"})
		return
	}

	var rt model.RefreshToken
	if err := db.DB.Where("token = ?", body.RefreshToken).First(&rt).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	if time.Now().After(rt.ExpiresAt) {
		// token expired, delete it
		db.DB.Delete(&rt)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token expired"})
		return
	}

	// load user
	var user model.User
	if err := db.DB.First(&user, rt.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	secret := os.Getenv("JWT_SECRET")
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))

	// rotate refresh token: delete old and create new
	db.DB.Delete(&rt)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}
	newRTStr := hex.EncodeToString(b)
	hrs := 168
	if v := os.Getenv("REFRESH_TOKEN_EXP_HOURS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			hrs = parsed
		}
	}
	newExpires := time.Now().Add(time.Duration(hrs) * time.Hour)
	newRT := model.RefreshToken{UserID: user.ID, Token: newRTStr, ExpiresAt: newExpires}
	if err := db.DB.Create(&newRT).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": tokenStr, "token_type": "Bearer", "refresh_token": newRTStr})
}
