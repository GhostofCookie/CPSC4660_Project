package main

import (
	"bufio"
	"encoding/json"
	"image"
	_ "image/jpeg"
	"io/ioutil"
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

	err := server.ConnectDatabases()
	if err != nil {
		log.Println(err)
		return
	}

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

	var response interface{}
	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	json.Unmarshal(body, &response)
	image := response.(map[string]interface{})["images"].(string)

	if image != "" {
		var err error
		var pgImg *Image
		retrievePostgres := func() error {
			pgImg, err = server.GetImage("postgres", strings.Split(image, ".")[0])
			return err
		}

		var myImg *Image
		retrieveMySQL := func() error {
			myImg, err = server.GetImage("mysql", strings.Split(image, ".")[0])
			return err
		}

		saveImage := func(imgData *Image, dir string) error {
			img, _, err := manager.BytesToImage(imgData.contents)
			if err != nil {
				return err
			}
			return manager.SaveImage(img, "static/temp/"+dir)
		}

		Benchmark("Postgres: Retrieve", retrievePostgres)
		Benchmark("MySQL: Retrieve", retrieveMySQL)

		err = saveImage(pgImg, "pg_"+image)
		if err != nil {
			log.Println(err)
		}
		err = saveImage(myImg, "my_"+image)
		if err != nil {
			log.Println(err)
		}
	}
}
