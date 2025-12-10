package handlers

import (
	"log"
	"net/http"
	"strconv"

	"messenger/internal/repositories"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProfileHandler struct {
	userRepo    *repositories.UserRepo
	contactRepo *repositories.ContactRepo
	contactSvc  *services.ContactService
	db          *gorm.DB
}

func NewProfileHandler(userRepo *repositories.UserRepo, contactRepo *repositories.ContactRepo, contactSvc *services.ContactService, db *gorm.DB) *ProfileHandler {
	return &ProfileHandler{
		userRepo:    userRepo,
		contactRepo: contactRepo,
		contactSvc:  contactSvc,
		db:          db,
	}
}

func (h *ProfileHandler) ShowProfile(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	currentUserID := userIDInterface.(uint)

	profileUserIDStr := c.Param("user_id")
	if profileUserIDStr == "" {
		profileUserIDStr = strconv.Itoa(int(currentUserID))
	}

	profileUserID64, err := strconv.ParseUint(profileUserIDStr, 10, 32)
	if err != nil {
		c.HTML(http.StatusBadRequest, "base.html", gin.H{
			"Page":  "profile",
			"Title": "Profile",
			"error": "Invalid user ID",
		})
		return
	}
	profileUserID := uint(profileUserID64)

	user, err := h.userRepo.FindByID(profileUserID)
	if err != nil {
		log.Println("Error fetching user:", err)
		c.HTML(http.StatusNotFound, "base.html", gin.H{
			"Page":  "profile",
			"Title": "Profile",
			"error": "User not found",
		})
		return
	}

	isOwn := profileUserID == currentUserID
	isContact, _ := h.contactSvc.IsContact(currentUserID, profileUserID) // Игнор err, false по умолчанию

	c.HTML(http.StatusOK, "base.html", gin.H{
		"Page":          "profile",
		"Title":         "Profile",
		"User":          user,
		"IsOwn":         isOwn,
		"IsContact":     isContact,
		"CurrentUserID": currentUserID,
	})
}

func (h *ProfileHandler) AddToContacts(c *gin.Context) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	currentUserID := userIDInterface.(uint)

	contactUserIDStr := c.Param("user_id")
	contactUserID64, err := strconv.ParseUint(contactUserIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	contactUserID := uint(contactUserID64)

	err = h.contactSvc.AddContact(currentUserID, contactUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Added to contacts"})
}
