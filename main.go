package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Source struct {
	ID   interface{} `json:"id"`
	Name string      `json:"name"`
}

type Article struct {
	Source      Source    `json:"source"`
	Author      string    `json:"author"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	URLToImage  string    `json:"urlToImage"`
	PublishedAt time.Time `json:"publishedAt"`
	Content     string    `json:"content"`
}

func (a *Article) FormatPublishedDate() string {
	year, month, day := a.PublishedAt.Date()
	return fmt.Sprintf("%v %d, %d", month, day, year)
}

type Results struct {
	Status       string    `json:"status"`
	TotalResults int       `json:"totalResults"`
	Articles     []Article `json:"articles"`
}

type Search struct {
	SearchKey  string
	NextPage   int
	TotalPages int
	Results    Results
}

func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}

	return s.NextPage - 1
}

func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

func (s *Search) IsLastPage() bool {
	return s.NextPage > s.TotalPages
}

var tmp = template.Must(template.ParseFiles("index.html"))
var apiKey *string
var maxArticles = 100

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmp.Execute(w, nil)
}

func serachHandler(w http.ResponseWriter, r *http.Request) {

	searchUrl, err := url.Parse(r.URL.String())

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
		return
	}

	params := searchUrl.Query()
	searchKey := params.Get("q")
	page := params.Get("page")

	if page == "" {
		page = "1"
	}

	search := &Search{}

	search.SearchKey = searchKey

	search.NextPage, err = strconv.Atoi(page)

	if err != nil {
		http.Error(w, "Unexpected server error", http.StatusInternalServerError)
		return
	}

	pageSize := 20

	endpoint := fmt.Sprintf("https://newsapi.org/v2/everything?q=%s&apiKey=%s&pageSize=%d&page=%d&sortBy=publishedAt&language=en", url.QueryEscape(search.SearchKey), *apiKey, pageSize, search.NextPage)

	resp, err := http.Get(endpoint)

	if err != nil {
		fmt.Println(err)
		http.Error(w, "Unexpected server error", http.StatusInternalServerError)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&search.Results)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	search.TotalPages = int(math.Ceil(float64(search.Results.TotalResults) / float64(pageSize)))

	if search.TotalPages*pageSize > maxArticles {
		search.TotalPages = int(math.Ceil(float64(maxArticles) / float64(pageSize)))
	}

	fmt.Println(search.TotalPages)

	if ok := !search.IsLastPage(); ok {
		search.NextPage++
	}

	err = tmp.Execute(w, search)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func main() {

	apiKey = flag.String("apikey", "", "Newsapi.org access key")

	flag.Parse()

	if *apiKey == "" {
		log.Fatal("apiKey must be set")
	}

	port := os.Getenv("PORT")

	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/search", serachHandler)

	err := http.ListenAndServe(":"+port, mux)

	if err != nil {
		log.Fatal(err)
	}
}
