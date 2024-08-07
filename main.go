package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"movinglake.com/haven/handler"
	"movinglake.com/haven/wrappers"
)

func main() {
	// Load dotenv.
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get vars from env.
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbHost := os.Getenv("DB_HOST")
	dbStr := fmt.Sprintf("host=%s user=%s dbname=%s password=%s", dbHost, dbUser, dbName, dbPass)

	// Create DB connection.
	db, err := wrappers.NewDB(dbStr)
	if err != nil {
		log.Fatal(err)
	}

	gin_mode := os.Getenv("GIN_MODE")
	if gin_mode != "" {
		gin.SetMode(gin.ReleaseMode)
	}

	nc := &handler.NotificationsConfig{
		SlackToken:     os.Getenv("SLACK_TOKEN"),
		SlackChannelID: os.Getenv("SLACK_CHANNEL"),
	}

	apiHandler := handler.NewHavenAPIHandler(db, nc)
	htmlHandler := handler.NewHavenHTMLHandler(db)

	r := gin.Default()
	apiHandler.RegisterRoutes(r)
	htmlHandler.RegisterRoutes(r, "templates/*", "web_resources")
	r.Run() // listen and serve on 0.0.0.0:8080
}
