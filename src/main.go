package main

import (
	"main/controllers"
	"main/db"

	"github.com/gin-gonic/gin"
)


func main() {
	database, _ := db.ConnectMySQL()
	controllers.InitAuth(database)
	r := gin.Default()

	r.Static("uploads", "./uploads")

	r.POST("/register", controllers.Register)
	r.Run(":8080")
}
