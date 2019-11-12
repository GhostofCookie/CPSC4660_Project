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

// Benchmark Checks the time elapsed when running a function.
func Benchmark(name string, fn func() error) {
	log.Println("=== Benchmark ", "\""+name+"\"", "===")
	start := time.Now()
	err := fn()
	if err != nil {
		log.Println("!!! Error:", err, "!!!")
	}
	elapsed := time.Since(start)
	log.Println("=== Time Elapsed:", elapsed)
}

func main() {
	manager.RegisterFormats()

	server = NewServer()
	server.HandleHTTP("/upload", UploadImage)
	server.HandleHTTP("/retrieve", RetrieveImage)

	server.BuildHTMLTemplate("static/index.html", "/", func(w http.ResponseWriter, r *http.Request) interface{} {
		return struct {
			OrigImg string
			PgImg   string
			MyImg   string
			Imgs    []string
		}{
			os.Getenv("OrigImg"),
			os.Getenv("PgImg"),
			os.Getenv("MyImg"),
			server.GetListImages(),
		}
	})

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

	storeMySQL := func() error {
		return server.StoreImage("mysql", name, format, int(handler.Size), bytes)
	}

	Benchmark("Postgres: Store", storePostgres)
	Benchmark("MySQL: Store", storeMySQL)

	os.Setenv("OrigImg", handler.Filename)
	os.Setenv("PgImg", "pg_"+os.Getenv("OrigImg"))
	os.Setenv("MyImg", "my_"+os.Getenv("OrigImg"))
	img, _, _ := manager.BytesToImage(bytes)
	manager.SaveImage(img, "static/temp/"+os.Getenv("OrigImg"))
}

// RetrieveImage Retrieves image from both Postgres and MySQL and saves them
// locally to be rendered to the web browser.
func RetrieveImage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	saveImage := func(imgData Image, dir string) {
		img, _, _ := manager.BytesToImage(imgData.contents)
		manager.SaveImage(img, "static/temp/"+dir)
	}

	log.Println(r.FormValue("images"))

	var err error
	var pgImg Image
	retrievePostgres := func() error {
		pgImg, err = server.GetImage("postgres", strings.Split(r.FormValue("images"), ".")[0])
		return err
	}

	var myImg Image
	retrieveMySQL := func() error {
		myImg, err = server.GetImage("mysql", strings.Split(r.FormValue("images"), ".")[0])
		return err
	}

	Benchmark("Postgres: Retrieve", retrievePostgres)
	Benchmark("MySQL: Retrieve", retrieveMySQL)

	saveImage(pgImg, os.Getenv("PgImg"))
	saveImage(myImg, os.Getenv("MyImg"))
}
