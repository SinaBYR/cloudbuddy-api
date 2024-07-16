package main

import (
	"cloudbuddy/internal/app/middleware"
	"cloudbuddy/internal/app/routes"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	cl "github.com/ostafen/clover/v2"
)

func main() {
	db, _ := cl.Open("clover-db")
	defer db.Close()

	if has, _ := db.HasCollection("images"); !has {
		db.CreateCollection("images")
	}
	if has, _ := db.HasCollection("users"); !has {
		db.CreateCollection("users")
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	images := r.Group("/v1/images")

	images.GET("", routes.GetAllImages(db))
	images.GET("/:id", routes.GetImageById(db))
	images.POST("", middleware.DecodeJwtMiddleware(db), routes.PostImage(db))
	images.PUT("/:id/like", routes.LikeImage(db))
	images.PUT("/:id/dislike", routes.DislikeImage(db))
	images.PUT(":id/changeTitle", middleware.DecodeJwtMiddleware(db), routes.ChangeImageTitle(db))
	images.DELETE("/:id", middleware.DecodeJwtMiddleware(db), routes.DeleteImage(db))

	auth := r.Group("/v1/auth")
	auth.POST("/signup", routes.Signup(db))
	auth.POST("/signin", routes.Signin(db))

	r.Run()
}
