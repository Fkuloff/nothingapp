// internal/handlers/auth.go
package handlers

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	authService *services.AuthService
	secret      []byte
}

func NewAuthHandler(authService *services.AuthService, secret []byte) *AuthHandler {
	return &AuthHandler{authService: authService, secret: secret}
}

func (h *AuthHandler) ShowRegister(c *gin.Context) {
	c.HTML(http.StatusOK, "base.html", gin.H{
		"Page":  "register",
		"Title": "Register",
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `form:"username" binding:"required,min=3,max=20"`
		Password string `form:"password" binding:"required,min=6"`
		Name     string `form:"name" binding:"required,min=2,max=50"`
		Phone    string `form:"phone" binding:"required"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":  "register",
			"Title": "Register",
			"error": err.Error(),
		})
		return
	}

	// Дополнительная валидация Phone (E.164 формат)
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(strings.TrimSpace(req.Phone)) {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":  "register",
			"Title": "Register",
			"error": "Invalid phone format (use +7XXXXXXXXXX)",
		})
		return
	}

	// Trim для чистоты
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)

	err := h.authService.Register(req.Username, req.Password, req.Name, req.Phone)
	if err != nil {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":  "register",
			"Title": "Register",
			"error": "Failed to register: " + err.Error(),
		})
		return
	}

	// Авто-логин после регистрации
	user, loginErr := h.authService.Login(req.Username, req.Password)
	if loginErr != nil {
		c.HTML(http.StatusInternalServerError, "base.html", gin.H{
			"Page":  "register",
			"Title": "Register",
			"error": "Registration successful, but login failed: " + loginErr.Error(),
		})
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString(h.secret)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "base.html", gin.H{
			"Page":  "register",
			"Title": "Register",
			"error": "Failed to generate token",
		})
		return
	}

	// Set cookie
	secure := false
	c.SetCookie("jwt_token", tokenString, int(time.Hour*24/time.Second), "/", "", secure, true)

	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "base.html", gin.H{
		"Page":  "login",
		"Title": "Login",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password" binding:"required"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":  "login",
			"Title": "Login",
			"error": err.Error(),
		})
		return
	}

	user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "base.html", gin.H{
			"Page":  "login",
			"Title": "Login",
			"error": "Invalid credentials",
		})
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString(h.secret)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "base.html", gin.H{
			"Page":  "login",
			"Title": "Login",
			"error": "Failed to generate token",
		})
		return
	}

	// Set cookie (secure: true in prod with HTTPS)
	secure := false // Set to true in production
	c.SetCookie("jwt_token", tokenString, int(time.Hour*24/time.Second), "/", "", secure, true)

	c.Redirect(http.StatusFound, "/")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("jwt_token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}
