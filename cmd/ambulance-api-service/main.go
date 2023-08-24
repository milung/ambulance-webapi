package main

import (
	_ "embed"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/milung/ambulance-webapi/api"
	"github.com/milung/ambulance-webapi/internal/ambulance_wl"
	"github.com/milung/ambulance-webapi/internal/db_service"
)

func main() {
	log.Printf("Server started")

	port := os.Getenv("AMBULANCE_API_PORT")
	if port == "" {
		port = "8080"
	}

	environment := os.Getenv("AMBULANCE_API_ENVIRONMENT")
	if !strings.EqualFold(environment, "production") { // case insensitive comparison
		gin.SetMode(gin.DebugMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())

	// setup context update  middleware
	dbService := db_service.NewMongoService(db_service.MongoServiceConfig{})
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("db_service", dbService)
		ctx.Next()
	})

	// request routings
	ambulance_wl.AddRoutes(engine)

	engine.GET("/openapi", api.HandleOpenApi)

	engine.Run(":" + port)
}
