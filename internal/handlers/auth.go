// internal/handlers/auth.go
package handlers

import (
	"net/http"
	"strconv"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) ShowRegister(c *gin.Context) {
	c.HTML(http.StatusOK, "base.html", gin.H{
		"Page":  "register",
		"Title": "Register",
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"error": "Username and password are required"})
		return
	}

	err := h.authService.Register(username, password)
	if err != nil {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":  "register",
			"Title": "Register",
			"error": "Username and password are required",
		})
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "base.html", gin.H{
		"Page":  "login",
		"Title": "Login",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{"error": "Username and password are required"})
		return
	}

	user, err := h.authService.Login(username, password)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Invalid credentials"})
		return
	}

	c.Redirect(http.StatusFound, "/chats?user_id="+strconv.Itoa(int(user.ID)))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.Redirect(http.StatusFound, "/login")
}
