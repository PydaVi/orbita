package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

// Confirmed against the real API before writing this: movies and TV
// shows use different field names for title/date (title/release_date
// vs. name/first_air_date), and poster_path is relative — needs this
// exact base to become a full URL (from GET /3/configuration).
const tmdbImageBase = "https://image.tmdb.org/t/p/w342"

type resolvedWork struct {
	Title     string
	PosterURL string
	Year      string
	Overview  string
}

func fetchFromTMDB(kind, id string) (resolvedWork, error) {
	apiKey := os.Getenv("TMDB_API_KEY")
	if apiKey == "" {
		return resolvedWork{}, fmt.Errorf("TMDB_API_KEY not set")
	}

	u := fmt.Sprintf("https://api.themoviedb.org/3/%s/%s?api_key=%s",
		kind, url.PathEscape(id), url.QueryEscape(apiKey))

	resp, err := http.Get(u)
	if err != nil {
		return resolvedWork{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resolvedWork{}, fmt.Errorf("TMDB returned %d for %s/%s", resp.StatusCode, kind, id)
	}

	var body struct {
		Title        string `json:"title"`
		Name         string `json:"name"`
		PosterPath   string `json:"poster_path"`
		ReleaseDate  string `json:"release_date"`
		FirstAirDate string `json:"first_air_date"`
		Overview     string `json:"overview"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return resolvedWork{}, err
	}

	title, date := body.Title, body.ReleaseDate
	if kind == "tv" {
		title, date = body.Name, body.FirstAirDate
	}

	year := ""
	if len(date) >= 4 {
		year = date[:4]
	}
	poster := ""
	if body.PosterPath != "" {
		poster = tmdbImageBase + body.PosterPath
	}

	return resolvedWork{Title: title, PosterURL: poster, Year: year, Overview: body.Overview}, nil
}

type searchResult struct {
	Provider  string `json:"provider"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Year      string `json:"year"`
	PosterURL string `json:"posterUrl,omitempty"`
}

// searchTMDB queries both the movie and TV search endpoints (same
// response shape as the detail endpoints, confirmed against the real
// API before writing this) since the person searching usually doesn't
// know or care which one it is — capped at 5 results per kind to keep
// the page small.
func searchTMDB(query string) ([]searchResult, error) {
	apiKey := os.Getenv("TMDB_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("TMDB_API_KEY not set")
	}

	var results []searchResult
	for _, kind := range []string{"movie", "tv"} {
		u := fmt.Sprintf("https://api.themoviedb.org/3/search/%s?query=%s&api_key=%s",
			kind, url.QueryEscape(query), url.QueryEscape(apiKey))

		resp, err := http.Get(u)
		if err != nil {
			return nil, err
		}
		var body struct {
			Results []struct {
				ID           int    `json:"id"`
				Title        string `json:"title"`
				Name         string `json:"name"`
				PosterPath   string `json:"poster_path"`
				ReleaseDate  string `json:"release_date"`
				FirstAirDate string `json:"first_air_date"`
			} `json:"results"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if decodeErr != nil {
			return nil, decodeErr
		}

		provider := "tmdb-movie"
		if kind == "tv" {
			provider = "tmdb-tv"
		}

		for i, r := range body.Results {
			if i >= 5 {
				break
			}
			title, date := r.Title, r.ReleaseDate
			if kind == "tv" {
				title, date = r.Name, r.FirstAirDate
			}
			year := ""
			if len(date) >= 4 {
				year = date[:4]
			}
			poster := ""
			if r.PosterPath != "" {
				poster = tmdbImageBase + r.PosterPath
			}
			results = append(results, searchResult{
				Provider:  provider,
				ID:        fmt.Sprintf("%d", r.ID),
				Title:     title,
				Year:      year,
				PosterURL: poster,
			})
		}
	}
	return results, nil
}

// resolveWork maps our Lexicon's provider values to the right TMDB
// endpoint. musicbrainz/open-library are declared in the Lexicon's
// knownValues but have no resolver yet — fails, and the caller falls
// back to showing the raw id (fail-open, same spirit as comum's cache).
func resolveWork(provider, id string) (resolvedWork, error) {
	switch provider {
	case "tmdb-movie":
		return fetchFromTMDB("movie", id)
	case "tmdb-tv":
		return fetchFromTMDB("tv", id)
	default:
		return resolvedWork{}, fmt.Errorf("no resolver yet for provider: %s", provider)
	}
}

// displayWork is what GET /shelf and GET /works/{provider}/{id} actually
// call. Cache-first; falls back to the raw "provider/id" string on any
// failure (unsupported provider, TMDB down, rate limited) instead of
// breaking the page — same fail-open spirit as comum's Redis cache.
func displayWork(db *sql.DB, provider, workID string) (title, posterURL, overview string) {
	if t, p, _, o, ok := getCachedWork(db, provider, workID); ok {
		return t, p, o
	}

	w, err := resolveWork(provider, workID)
	if err != nil {
		log.Printf("could not resolve %s/%s, showing raw id: %v", provider, workID, err)
		return fmt.Sprintf("%s/%s", provider, workID), "", ""
	}

	if err := setCachedWork(db, provider, workID, w.Title, w.PosterURL, w.Year, w.Overview); err != nil {
		log.Printf("failed to cache %s/%s: %v", provider, workID, err)
	}
	return w.Title, w.PosterURL, w.Overview
}
