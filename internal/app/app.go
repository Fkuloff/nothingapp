// internal/app/app.go
package app

import (
	"log"
	"os"

	"messenger/internal/config"
	"messenger/internal/handlers"
	"messenger/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Run() {
	cfg := config.LoadConfig()

	var err error
	DB, err = gorm.Open(postgres.Open(cfg.DBURL), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	err = DB.AutoMigrate(&models.User{}, &models.Chat{}, &models.Message{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	router := gin.Default()

	// Статические файлы
	router.Static("/static", "./static")

	// Шаблоны
	router.LoadHTMLGlob("templates/*.html")

	handlers.SetupRoutes(router, DB)

	router.Run(":" + os.Getenv("PORT"))
}
