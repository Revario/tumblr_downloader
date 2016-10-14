/*
   This is a go program to download pictures from a tumblr blog page.

   To Build:

       go build -o tumblr-download tumblr-download.go

   To Run:

       # download the photos on the first page of tumblr blog
       # http://jnightscape.tumblr.com
       tumblr-download http://jnightscape.tumblr.com

       # download the 2nd page of photos
       tumblr-download -page 2 http://jnightscape.tumblr.com

       # examine the raw JSON output from REST request on tumblr blog
       # (helpful for debugging)
       tumblr-download -raw http://jnightscape.tumblr.com

   Note:

       Pictures will download to the current working directory where
       you're running the command.
*/
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

var pageCounter = 0

type Post struct {
	Id       string `json:"id"`
	Url      string `json:"url"`
	Class    string `json:"type"`
	Date     string `json:"date"`
	Caption  string `json:"photo-caption"`
	PhotoUrl string `json:"photo-url-1280"`
}

type TumblrLog struct {
	Title string `json:"title"`
	Name  string `json:"name"`
}

type Tumblr struct {
	Blog          TumblrLog `json:"tumblelog"`
	Posts         []Post    `json:"posts"`
	NumberOfPosts int       `json:"posts-total"`
}

func NewTumblr(url string, page int, silent bool) Tumblr {
	contents := GetJson(url, page, silent)

	var t Tumblr
	json.Unmarshal(contents, &t)
	return t
}

func GetJson(url string, page int, silent bool) []byte {
	contents := restRequest(url, page, silent)
	contents = filterContent(contents, "var tumblr_api_read = ", "", 1)
	contents = filterContent(contents, ";", "", -1)
	return contents
}

func filterContent(data []byte, orig string, target string, n int) []byte {
	c := string(data)
	c = strings.Replace(c, orig, target, n)
	return []byte(c)
}

func restRequest(url string, page int, silent bool) []byte {
	if page != 1 {
		p := (page - 1) * 20
		url = fmt.Sprintf("%s/api/read/json?start=%d", url, p)
	} else {
		url = fmt.Sprintf("%s/api/read/json", url)
	}

	if !silent {
		fmt.Println("REST Request url: ", url)
	}

	resp, err := http.Get(url)
	defer resp.Body.Close()

	if err != nil {
		log.Fatal("Trouble making REST GET request!")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Trouble reading JSON response body!")
	}

	return contents
}

func displayRawJson(contents []byte) {
	var out bytes.Buffer
	err := json.Indent(&out, contents, "", "    ")
	if err != nil {
		log.Fatal("Trouble with json indent!", err)
	}
	fmt.Println("")
	fmt.Println("---")
	out.WriteTo(os.Stdout)
	os.Exit(0)
}

func (t Tumblr) DownloadImages(silent bool) {

	if silent {
		for _, post := range t.Posts {
			if post.Class != "photo" {
				continue
			}
			post.downloadImage()
		}
	} else {
		for i, post := range t.Posts {
			fmt.Println("Post # ", i)
			fmt.Println(" ---> Caption: ", post.Caption)
			fmt.Println(" ---> Url    : ", post.PhotoUrl)
			if post.Class != "photo" {
				fmt.Println(" ---> SKIPPING (not photo post)")
				continue
			}
			post.downloadImage()
			fmt.Println()
		}
	}

}

func (p Post) downloadImage() {
	resp, err := http.Get(p.PhotoUrl)
	defer resp.Body.Close()

	if err != nil {
		log.Fatal("Trouble making GET photo request!")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Trouble reading response body!")
	}

	filename := path.Base(p.PhotoUrl)
	if filename == "" {
		log.Fatalf("Trouble deriving file name for %s", p.PhotoUrl)
	}

	err = ioutil.WriteFile(filename, contents, 0644)
	if err != nil {
		log.Fatal("Trouble creating file! -- ", err)
	}
}

func main() {
	pagePtr := flag.Int("page", 1, "blog page to download")
	rawJsonPtr := flag.Bool("raw", false, "dump raw json output for debugging")
	allPtr := flag.Bool("all", false, "downloads all images")
	flag.Parse()

	url := strings.TrimSuffix(flag.Arg(0), "/")

	if url == "" {
		fmt.Fprint(os.Stderr, "Please supply a tumblr url!\n")
		fmt.Fprintf(os.Stderr, "usage: %s [options] url\n", os.Args[0])
		os.Exit(0)
	}

	if *rawJsonPtr == true {
		contents := GetJson(url, *pagePtr, false)
		displayRawJson(contents)
		os.Exit(0)
	}

	if *allPtr == true {
		t := NewTumblr(url, *pagePtr, true)
		pages := t.NumberOfPosts / 20
		if t.NumberOfPosts%20 != 0 {
			pages++
		}

		for i := 1; i < pages; i++ {
			t := NewTumblr(url, i, true)
			t.DownloadImages(true)
			pageCounter++
			time.Sleep(time.Duration(10) * time.Second)
		}

		fmt.Printf("Done! %d of %d pages downloaded", pageCounter, pages)
		os.Exit(0)

	} else {
		t := NewTumblr(url, *pagePtr, false)
		fmt.Println("Blog Title: ", t.Blog.Title)
		fmt.Println("Number of Posts: ", t.NumberOfPosts)
		t.DownloadImages(false)
		os.Exit(0)
	}

}
