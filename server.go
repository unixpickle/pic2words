package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var AssetsPath string
var WordList []string

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run server.go <port>")
		os.Exit(1)
	}

	findAssets()
	readWordList()

	http.Handle("/pic2words", http.HandlerFunc(handlePic2Words))
	http.Handle("/words2pic", http.HandlerFunc(handleWords2Pic))
	http.Handle("/", http.HandlerFunc(handleHome))

	http.ListenAndServe(":"+os.Args[1], http.DefaultServeMux)
}

func findAssets() {
	_, filename, _, _ := runtime.Caller(0)
	AssetsPath = path.Join(path.Dir(filename), "assets")
}

func readWordList() {
	contents, err := ioutil.ReadFile(path.Join(AssetsPath, "words.txt"))
	if err != nil {
		panic(err)
	}
	WordList = strings.Split(string(contents), "\n")
	if len(WordList) != 0x10000 {
		panic("invalid wordlist length")
	}
}

func handlePic2Words(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	resp, err := http.Get(r.Form.Get("url"))
	if err != nil {
		http.ServeFile(w, r, path.Join(AssetsPath, "bad_url.html"))
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.ServeFile(w, r, path.Join(AssetsPath, "bad_url.html"))
		return
	}

	words := dataToWords(body)
	result := strings.Join(words, " ")
	intro := "There were " + strconv.Itoa(len(words)) + " words." +
		" Here they are: <br><br>"
	w.Write([]byte("<!doctype html><html><body>" + intro + result +
		"</body></html>"))
}

func handleWords2Pic(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	words := strings.Split(strings.TrimSpace(r.Form.Get("words")), " ")
	data := wordsToData(words)

	// Check if the data is a valid image in itself.
	buffer := bytes.NewBuffer(data)
	_, _, err := image.Decode(buffer)
	if err == nil {
		w.Write(data)
	} else {
		if len(data) < 0x100 {
			w.Write([]byte("The image you entered would probably be too small to see."))
		} else {
			png.Encode(w, rawBitmapToImage(data))
		}
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join(AssetsPath, "index.html"))
}

func dataToWords(data []byte) []string {
	var words []string
	if len(data)%2 == 0 {
		words = []string{"Even"}
	} else {
		words = []string{"Odd"}
	}
	for i := 0; i < len(data); i += 2 {
		x1 := int(data[i])
		var x2 int
		if i+1 < len(data) {
			x2 = int(data[i+1])
		}
		num := x1 + x2*0x100
		words = append(words, WordList[num])
	}
	return words
}

func wordsToData(words []string) []byte {
	if len(words) == 0 {
		return []byte("Invalid words.")
	}
	even := strings.ToLower(words[0]) == "even"
	data := []byte{}
	for i, word := range words {
		if i == 0 {
			continue
		}
		word = strings.Trim(word, ".,;! \n")
		idx := sort.SearchStrings(WordList, strings.ToLower(word))
		lower := idx % 0x100
		upper := idx / 0x100
		data = append(data, byte(lower), byte(upper))
	}
	if !even && len(data) > 0 {
		return data[:len(data)-1]
	}
	return data
}

func rawBitmapToImage(data []byte) image.Image {
	size := int(math.Ceil(math.Sqrt(float64(len(data) / 2))))
	bitmap := image.NewRGBA(image.Rect(0, 0, size, size))
	for i := 0; i < size*size; i++ {
		x := i % size
		y := i / size
		idx := (i % (len(data)/3)) * 3
		color := color.RGBA{uint8(data[idx]), uint8(data[idx+1]),
			uint8(data[idx+2]), 0xff}
		bitmap.Set(x, y, color)
	}
	return bitmap
}
