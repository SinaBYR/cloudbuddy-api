package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	cl "github.com/ostafen/clover/v2"
	q "github.com/ostafen/clover/v2/query"
)

type Image struct {
	UUID      string    `clover:"uuid"`
	Url       string    `clover:"url"`
	Likes     int64     `clover:"likes"`
	CreatedAt time.Time `clover:"created_at"`
}

func main() {
	db, _ := cl.Open("clover-db")
	defer db.Close()

	if has, _ := db.HasCollection("images"); !has {
		db.CreateCollection("images")
	}

	r := gin.Default()

	images := r.Group("/v1/images")

	images.GET("/", getAllImages(db))

	r.Run()
}

func getAllImages(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		docs, err := db.FindAll(q.NewQuery("images"))

		fmt.Println(docs)

		if err != nil {
			if err == cl.ErrCollectionNotExist {
				c.JSON(http.StatusNotFound, gin.H{
					"message": err.Error(),
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "wtf",
				})
			}

			return
		}

		c.JSON(http.StatusOK, docs)
	}
}
