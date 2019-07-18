package main

import (
	"bytes"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
)

func GetImage(url string) (image.Image, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[GetImage][HTTPGet] Error: %v", err)
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[GetImage][ReadBytes] Error: %v", err)
		return nil, 0, err
	}
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		log.Printf("[GetImage][Decode] Error: %v", err)
		return nil, 0, err
	}
	return img, len(b), nil
}

func Resize(I image.Image) (image.Image, int) {
	I = resize.Resize(uint(I.Bounds().Max.X/2), 0, I, resize.NearestNeighbor)
	b, _ := Encode(I)
	return I, len(b)
}

func Encode(I image.Image) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, I, nil)
	if err != nil {
		log.Printf("[Encode] Error: %v", err)
	}
	return buf.Bytes(), err
}
