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

type ImgSize struct {
	postfix string
	size    int
}

var imgSizesAvatar = []ImgSize{{"_xl", 1000}, {"_d", 300}, {"_s", 100}}
var imgSizesPost = []ImgSize{{"_xl", 1000}}

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
	_, err := file.Read(buff)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	filetype := http.DetectContentType(buff)
	if filetype != "image/jpeg" && filetype != "image/png" {
		return "", fmt.Errorf("format error")
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	// DB
	storeId := uuid.NewString()

	_, err = db.Query("insert into photos (store_id, user_id, date) values ($1, $2, $3)", storeId, rl.id, date)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	// Storage
	var imgFile image.Image
	var buf io.Reader

	switch filetype {
	case "image/jpeg":
		imgFile, err = jpeg.Decode(file, &jpeg.DecoderOptions{})
		if err != nil {
			fmt.Println(err)
			return "", err
		}

		bufTemp := bytes.Buffer{}
		if err := png.Encode(&bufTemp, imgFile); err != nil {
			fmt.Println(err)
			return "", err
		}

		buf = &bufTemp
	case "image/png":
		buf = file

		imgFile, err = png.Decode(file)
		if err != nil {
			fmt.Println("png decode ", err)
			return "", err
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			fmt.Println(err)
			return "", err
		}
	}

	// Original
	reqSavePhoto(storeId+".png", buf)

	// Resize
	var imgSizes *[]ImgSize = &imgSizesPost
	if avatar {
		imgSizes = &imgSizesAvatar
	}

	err = saveImageSamples(storeId, imgFile, *imgSizes)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return storeId, nil
}

func saveImageSamples(storeId string, img image.Image, sizes []ImgSize) error {
	b := img.Bounds()
	imgWidth := b.Max.X
	imgHeight := b.Max.Y

	for _, size := range sizes {
		args := [2]int{size.size, 0}
		if imgHeight > imgWidth {
			args[0] = 0
			args[1] = size.size
		}

		src := imaging.Resize(img, args[0], args[1], imaging.Lanczos)
		buf := &bytes.Buffer{}
		if err := png.Encode(buf, src); err != nil {
			fmt.Println(size.postfix, " encode ", err)
			return err
		}

		name := storeId + size.postfix
		reqSavePhoto(name+".png", buf)
	}

	return nil
}

func cleanPhotos(photoIds []string, avatar bool) ([]string, *[]error) {
	query := "update photos set deleted=true where store_id=ANY($1) RETURNING store_id"
	rows, err := db.Query(query, pq.Array(photoIds))
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

		sizes := &imgSizesPost
		if avatar {
			sizes = &imgSizesAvatar
		}

		for _, size := range *sizes {
			rmObject(photoId + size.postfix + ".png")
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
