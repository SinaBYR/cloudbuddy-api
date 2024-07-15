package routes

import (
	"cloudbuddy/internal/pkg"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	cl "github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	q "github.com/ostafen/clover/v2/query"
)

var ImagesCount = -1

func GetImageById(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		doc, err := db.FindFirst(q.NewQuery("images").Where(q.Field("_id").Eq(id)))

		if doc == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "cloud not found :(",
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An unexpected error occured",
			})
			return
		}

		c.JSON(http.StatusOK, pkg.Image{
			UUID:      doc.Get("_id").(string),
			Title:     doc.Get("title").(string),
			Url:       doc.Get("url").(string),
			Likes:     doc.Get("likes").(int64),
			UserId:    doc.Get("user_id").(string),
			CreatedAt: doc.Get("created_at").(time.Time),
		})
	}
}

// returns all images.
// limit default is 5.
// offset default is 0.
func GetAllImages(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		offsetQuery := c.Query("offset")
		offset, err := strconv.Atoi(offsetQuery)
		limit := 5
		if err != nil {
			offset = 0
		}
		docs, err := db.FindAll(q.NewQuery("images").Sort(q.SortOption{Field: "created_at", Direction: -1}).Skip(offset).Limit(limit))
		if ImagesCount == -1 {
			count, err := db.Count(q.NewQuery("images"))
			if err != nil {
				log.Println(err)
			} else {
				ImagesCount = count
			}
		}

		var images []pkg.Image = []pkg.Image{}

		for _, doc := range docs {
			images = append(images, pkg.Image{
				UUID:      doc.Get("_id").(string),
				Title:     doc.Get("title").(string),
				Url:       doc.Get("url").(string),
				Likes:     doc.Get("likes").(int64),
				UserId:    doc.Get("user_id").(string),
				CreatedAt: doc.Get("created_at").(time.Time),
			})
		}

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

		c.JSON(http.StatusOK, gin.H{
			"images": images,
			"count":  ImagesCount,
		})
	}
}

func PostImage(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		title := c.PostForm("title")
		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusNotAcceptable, gin.H{
				"message": err.Error(),
			})
			return
		}
		if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"message": "Uploaded file must be an image",
			})
			return
		}

		user, exists := c.Get("user")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		userId, ok := user.(*document.Document).Get("_id").(string)

		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Something went wrong on the server",
			})
			return
		}

		doc := document.NewDocument()
		doc.Set("title", title)
		doc.Set("url", "")
		doc.Set("likes", 0)
		doc.Set("user_id", userId)
		doc.Set("created_at", time.Now())
		docId, err := db.InsertOne("images", doc)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occured while creating a new image record",
			})
			return
		}

		err = pkg.UploadToBucket(file, docId)
		if err != nil {
			log.Println(err)
			innerErr := db.DeleteById("images", docId)
			if innerErr != nil {
				log.Println(innerErr)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "An error occured while deleting temporary created image record",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occured while uploading image to the bucket",
			})
			return
		}

		err = godotenv.Load()
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occured",
			})
			return
		}

		url := strings.Join([]string{
			os.Getenv("BUCKET_ENDPOINT") + "/" + os.Getenv("BUCKET_NAME"),
			"/cloudbuddy/",
			docId,
			"-",
			file.Filename,
		}, "")

		err = db.UpdateById("images", docId, func(doc *document.Document) *document.Document {
			doc.Set("url", url)
			return doc
		})

		if err != nil {
			if err == cl.ErrDocumentNotExist {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Image upload failed",
				})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "An error occured",
				})
				return
			}
		}

		ImagesCount += 1

		err = db.UpdateById("users", userId, func(doc *document.Document) *document.Document {
			newImageId := doc.Get("_id").(string)
			images := doc.Get("images").([]string)
			images = append(images, newImageId)
			doc.Set("images", images)
			return doc
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occured",
			})
			return
		}

		c.JSON(http.StatusCreated, pkg.Image{
			UUID:      doc.Get("_id").(string),
			Title:     doc.Get("title").(string),
			Url:       url,
			UserId:    doc.Get("user_id").(string),
			Likes:     doc.Get("likes").(int64),
			CreatedAt: doc.Get("created_at").(time.Time),
		})
	}
}

func LikeImage(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		err := db.UpdateById("images", id, func(doc *document.Document) *document.Document {
			doc.Set("likes", doc.Get("likes").(int64)+1)
			return doc
		})

		if err != nil {
			if err == cl.ErrDocumentNotExist {
				c.JSON(http.StatusNotFound, gin.H{
					"message": "cloud not found :(",
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "wtf",
				})
			}

			return
		}

		c.JSON(http.StatusNoContent, nil)
	}
}

func DislikeImage(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")
		err := db.UpdateById("images", id, func(doc *document.Document) *document.Document {
			doc.Set("likes", doc.Get("likes").(int64)-1)
			return doc
		})

		if err != nil {
			if err == cl.ErrDocumentNotExist {
				c.JSON(http.StatusNotFound, gin.H{
					"message": "cloud not found :(",
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "wtf",
				})
			}

			return
		}

		c.JSON(http.StatusNoContent, nil)
	}
}

func DeleteImage(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		id := c.Param("id")

		user, exists := c.Get("user")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			return
		}

		userId, ok := user.(*document.Document).Get("_id").(string)

		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Something went wrong on the server",
			})
			return
		}

		image, err := db.FindFirst(q.NewQuery("images").Where(q.Field("_id").Eq(id).And(q.Field("user_id").Eq(userId))))
		if image == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "image not found",
			})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "something went wrong on the server",
			})
			return
		}

		// TODO delete image from bucket

		err = db.DeleteById("images", image.ObjectId())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "something went wrong on the server while deleting the image",
			})
			return
		}

		ImagesCount -= 1

		err = db.UpdateById("images", id, func(doc *document.Document) *document.Document {
			images := doc.Get("images").([]string)
			images = pkg.RemoveByValue(images, image.ObjectId())
			doc.Set("images", images)
			return doc
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "An error occured",
			})
			return
		}

		c.JSON(http.StatusNoContent, nil)
	}
}
