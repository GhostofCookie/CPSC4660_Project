package main

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"log"
	"os"
)

type ImageManager struct {
}

type Image struct {
	name     string
	format   string
	size     int
	contents []byte
}

// RegisterFormats Registers all the image formats that can be used.
func (im *ImageManager) RegisterFormats() {
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
}

// OpenImage Opens an image file.
func (im *ImageManager) OpenImage(filename string) (image.Image, string, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		return nil, "", err
	}
	img, format, err := image.Decode(file)
	if err != nil {
		log.Println(err)
		return nil, "", err
	}
	return img, format, nil
}

// ImageToBytes convert a given image into a byte array.
func (im *ImageManager) ImageToBytes(img image.Image, format string) ([]byte, error) {
	buffer := new(bytes.Buffer)

	var err error
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(buffer, img, nil)
		break
	case "png":
		err = png.Encode(buffer, img)
		break
	}

	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// BytesToImage convert a byte array into an image.
func (im *ImageManager) BytesToImage(imgBytes []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(imgBytes))

	if err != nil {
		return nil, "", err
	}
	return img, format, err
}

// SaveImage Saves an image as a file.
func (im *ImageManager) SaveImage(img image.Image, newFile string) error {
	out, err := os.Create(newFile)
	if err != nil {
		return err
	}

	err = png.Encode(out, img)
	if err != nil {
		return err
	}
	defer out.Close()
	return nil
}