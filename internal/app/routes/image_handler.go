package routes

import (
	"cloudbuddy/internal/pkg"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	cl "github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	q "github.com/ostafen/clover/v2/query"
)

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

		c.JSON(http.StatusOK, pkg.Image{
			UUID:      doc.Get("_id").(string),
			Url:       doc.Get("url").(string),
			Likes:     doc.Get("likes").(int64),
			CreatedAt: doc.Get("created_at").(time.Time),
		})
	}
}

func GetAllImages(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		docs, err := db.FindAll(q.NewQuery("images"))

		var images []pkg.Image

		for _, doc := range docs {
			images = append(images, pkg.Image{
				UUID:      doc.Get("_id").(string),
				Url:       doc.Get("url").(string),
				Likes:     doc.Get("likes").(int64),
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

		c.JSON(http.StatusOK, images)
	}
}

func PostImage(db *cl.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
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

		doc := document.NewDocument()
		doc.Set("url", "")
		doc.Set("likes", 0)
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
			os.Getenv("BUCKET_ENDPOINT"),
			os.Getenv("BUCKET_NAME"),
			"cloudbuddy",
			file.Filename,
		}, "/")

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

		c.JSON(http.StatusCreated, pkg.Image{
			UUID:      doc.Get("_id").(string),
			Url:       doc.Get("url").(string),
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
