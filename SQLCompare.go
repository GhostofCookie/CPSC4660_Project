package main

import (
	"bufio"
	"image"
	_ "image/jpeg"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var manager ImageManager
var server *Server
var logger *os.File

func main() {
	manager.RegisterFormats()

	server = NewServer()
	server.HandleHTTP("/upload", UploadImage)

	server.BuildHTMLTemplate("static/index.html", "/", func(w http.ResponseWriter, r *http.Request) interface{} { return struct{}{} })

	server.ConnectDatabases()
	server.ConnectHTTP(80)
}

// UploadImage Handles the uploading of an image from the web client.
func UploadImage(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	imageFile, handler, err := r.FormFile("image")
	if err != nil {
		log.Println("!> Error Retrieving Image:", err)
	}
	defer imageFile.Close()

	log.Println("Uploaded Image: ", handler.Filename)
	log.Println("Image Size: ", handler.Size)
	name := strings.Split(handler.Filename, ".")[0]
	format := strings.SplitAfter(handler.Filename, ".")[1]

	rdr := bufio.NewReader(imageFile)
	tempImg, _, _ := image.Decode(rdr)

	bytes, err := manager.ImageToBytes(tempImg, format)
	if err != nil {
		log.Println("!> Error Reading Image:", err)
	}
	defer imageFile.Close()

	storePostgres := func() error {
		return server.StoreImage("postgres", name, format, int(handler.Size), bytes)
	}
	Benchmark("Postgres: Store", storePostgres)
	testImg, _ := server.GetImage("postgres", "connected")

	storeMySQL := func() error {
		return server.StoreImage("mysql", name, format, int(handler.Size), bytes)
	}
	Benchmark("MySQL: Store", storeMySQL)
	server.GetImage("mysql", "witcher_test")

	img, _, _ := manager.BytesToImage(testImg.contents)
	manager.SaveImage(img, "test1."+format)

}

// Benchmark Checks the time elapsed when running a function.
func Benchmark(name string, fn func() error) {
	log.Println("=== Benchmark ", "\""+name+"\"", "===")
	start := time.Now()
	err := fn()
	if err != nil {
		log.Println("!!! Error:", err, "!!!")
	}
	elapsed := time.Since(start)
	log.Println("=== Time Elapsed:", elapsed, "===")
}
