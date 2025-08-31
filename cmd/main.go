package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/routes"
)

func main() {
	config.LoadConfig()

	r := gin.Default()

	routes.SetupRoutes(r)

	r.Run(":" + config.PORT)

	fmt.Printf("Server starting on port %s\n", config.PORT)

	// Start server
	log.Fatal(r.Run(":" + config.PORT))
}
