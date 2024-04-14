package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/pixiv/go-libjpeg/jpeg"
)

var photoBuffer = NewPhotoBuffer()

type PhotoBuffer struct {
	mx sync.Mutex
	m  map[string][]byte
}

func (c *PhotoBuffer) Load(key string) ([]byte, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()
	val, ok := c.m[key]
	return val, ok
}

func (c *PhotoBuffer) Store(key string, value []byte) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.m[key] = value
}

func (c *PhotoBuffer) Delete(key string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	delete(c.m, key)
}

func NewPhotoBuffer() *PhotoBuffer {
	return &PhotoBuffer{
		m: make(map[string][]byte),
	}
}

func servStorage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check photo buffer
	if photo, ok := photoBuffer.Load(id); ok {
		w.Header().Set("Cache-Control", "max-age=25920000")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "image/png")

		io.Copy(w, bytes.NewReader(photo))
	} else {
		object := getObject(id)

		w.Header().Set("Cache-Control", "max-age=25920000")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "image/png")

		c, _ := io.Copy(w, object)
		defer object.Close()

		if c == 0 {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func loadImage(file multipart.File, rl role, date time.Time, avatar bool) (string, error) {
	// File format validation
	buff := make([]byte, 512)
	_, err := (file).Read(buff)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	filetype := http.DetectContentType(buff)
	if filetype != "image/jpeg" && filetype != "image/png" {
		return "", fmt.Errorf("format error")
	}

	_, err = (file).Seek(0, io.SeekStart)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	storeId := uuid.NewString()

	rows, err := db.Query("insert into photos (store_id, user_id, date) values ($1, $2, $3) RETURNING id", storeId, rl.id, date)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	var photoId int
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&photoId)
		if err != nil {
			fmt.Println(err)
			return "", err
		}

	} else {
		return "", fmt.Errorf("no photo returning")
	}

	if filetype == "image/jpeg" {
		img, err := jpeg.Decode(file, &jpeg.DecoderOptions{})
		if err != nil {
			fmt.Println(err)
			return "", err
		}

		buf := &bytes.Buffer{}
		if err := png.Encode(buf, img); err != nil {
			fmt.Println(err)
			return "", err
		}

		reqSavePhoto(storeId+".png", buf)

		if avatar {
			err = saveImageSamples(storeId, img)
			if err != nil {
				fmt.Println(err)
				return "", err
			}
		}
	} else if filetype == "image/png" {
		reqSavePhoto(storeId+".png", file)

		if avatar {
			_, err = (file).Seek(0, io.SeekStart)
			if err != nil {
				fmt.Println(err)
				return "", err
			}
			img, err := png.Decode(file)
			if err != nil {
				fmt.Println("png decode ", err)
				return "", err
			}

			err = saveImageSamples(storeId, img)
			if err != nil {
				fmt.Println(err)
				return "", err
			}
		}
	}

	return storeId, nil
}

func saveImageSamples(storeId string, img image.Image) error {
	// Resize [D size]
	src := imaging.Resize(img, 300, 0, imaging.Lanczos)
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, src); err != nil {
		fmt.Println("d encode ", err)
		return err
	}

	name := storeId + "_d"
	reqSavePhoto(name+".png", buf)

	// Resize [S size]
	src = imaging.Resize(img, 100, 0, imaging.Lanczos)
	buf = &bytes.Buffer{}
	if err := png.Encode(buf, src); err != nil {
		fmt.Println(err)
		return err
	}

	name = storeId + "_s"
	reqSavePhoto(name+".png", buf)

	// fmt.Println(name)

	return nil
}

func cleanPhotos(photoIds []string, avatar bool) ([]string, *[]error) {
	rows, err := db.Query("update photos set deleted=true where store_id=ANY($1) RETURNING store_id", pq.Array(photoIds))
	if err != nil {
		fmt.Println(err)
		return []string{}, &[]error{err}
	}
	defer rows.Close()

	var cleanedIds []string
	var errs []error

	for rows.Next() {
		var photoId string
		err := rows.Scan(&photoId)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		cleanedIds = append(cleanedIds, photoId)

		// Clean photo from storage

		rmObject(photoId + ".png")

		if avatar {
			rmObject(photoId + "_d.png")
			rmObject(photoId + "_s.png")
		}
	}

	if len(errs) == 0 {
		return cleanedIds, nil
	}

	return cleanedIds, &errs
}

func reqSavePhoto(name string, r io.Reader) {
	buf, _ := io.ReadAll(r)

	photoBuffer.Store(name, buf)
	go putObject(name, buf)
}

func putObject(name string, b []byte) {
	r := bytes.NewReader(b)
	info, err := minioClient.PutObject(context.Background(), config.S3.Buckets.Photo, name, r, int64(r.Size()), minio.PutObjectOptions{ContentType: "image/png"})
	photoBuffer.Delete(name)
	if err != nil {
		log.Fatalln("put", err, info)
	} else {
		photoBuffer.Delete(name)
	}
}

func getObject(name string) *minio.Object {
	object, err := minioClient.GetObject(context.Background(), config.S3.Buckets.Photo, name, minio.GetObjectOptions{})
	if err != nil {
		log.Println(err)
		return nil
	}

	return object
}

func rmObject(name string) {
	photoBuffer.Delete(name)

	opts := minio.RemoveObjectOptions{
		GovernanceBypass: true,
	}
	err := minioClient.RemoveObject(context.Background(), config.S3.Buckets.Photo, name, opts)
	if err != nil {
		log.Println(err)
		return
	}
}
