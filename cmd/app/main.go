package main

import (
	"cloudbuddy/internal/app/routes"

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

	images := r.Group("/v1/images")

	images.GET("/", routes.GetAllImages(db))
	images.GET("/:id", routes.GetImageById(db))
	images.POST("/", routes.PostImage(db))
	images.PUT("/:id/like", routes.LikeImage(db))
	images.PUT("/:id/dislike", routes.DislikeImage(db))

	auth := r.Group("/v1/auth")
	auth.POST("/signup", routes.Signup(db))

	r.Run()
}
