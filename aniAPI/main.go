package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

const url = "https://graphql.anilist.co"

type Title struct {
	Romaji string `json:"romaji"`
}

type Staff struct {
	Nodes []struct {
		Name struct {
			Full string `json:"full"`
		} `json:"name"`
	} `json:"nodes"`
}

type Media struct {
	ID          int      `json:"id"`
	Title       Title    `json:"title"`
	Description string   `json:"description"`
	Genres      []string `json:"genres"`
	Status      string   `json:"status"`
	Chapters    int      `json:"chapters"`
	Staff       Staff    `json:"staff"`
}

type Page struct {
	Media []Media `json:"media"`
}

type Response struct {
	Data struct {
		Page Page `json:"Page"`
	} `json:"data"`
}

type Manga struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"`
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
}

func fetchPage(page int, perPage int) ([]Media, error) {
	query := `
    query ($page: Int, $perPage: Int) {
      Page(page: $page, perPage: $perPage) {
        media(type: MANGA) {
          id
          title { romaji }
          description
          genres
          status
          chapters
          staff { nodes { name { full } } }
        }
      }
    }`

	variables := map[string]interface{}{
		"page":    page,
		"perPage": perPage,
	}

	reqBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	jsonData, _ := json.Marshal(reqBody)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var result Response
	json.Unmarshal(body, &result)

	return result.Data.Page.Media, nil
}

func main() {
	perPage := 50
	var allManga []Manga

	// Loop through exactly 10 pages
	for page := 1; page <= 40; page++ {
		media, err := fetchPage(page, perPage)
		if err != nil {
			panic(err)
		}

		for _, m := range media {
			author := ""
			if len(m.Staff.Nodes) > 0 {
				author = m.Staff.Nodes[0].Name.Full
			}
			allManga = append(allManga, Manga{
				ID:            fmt.Sprintf("%d", m.ID),
				Title:         m.Title.Romaji,
				Author:        author,
				Genres:        m.Genres,
				Status:        m.Status,
				TotalChapters: m.Chapters,
				Description:   m.Description,
			})
		}
		fmt.Printf("Fetched page %d with %d entries\n", page, len(media))
	}

	// Save all results to JSON file
	file, _ := os.Create("manga.json")
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(allManga)

	fmt.Println("Saved manga.json with", len(allManga), "entries")
}
