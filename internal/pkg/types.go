package pkg

import "time"

type Image struct {
	UUID      string    `clover:"_id" json:"uuid"`
	Url       string    `clover:"url" json:"image_url"`
	Likes     int64     `clover:"likes" json:"likes"`
	CreatedAt time.Time `clover:"created_at" json:"created_at"`
}
