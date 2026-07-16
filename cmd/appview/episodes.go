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

type season struct {
	Number       int
	Name         string
	EpisodeCount int
}

type episode struct {
	Number   int
	Name     string
	Overview string
	AirDate  string
}

// Cache tables, same disposable spirit as work_cache: dropping either
// just means the next page view re-fetches and re-fills them.
const seasonCacheSchema = `
CREATE TABLE IF NOT EXISTS season_cache (
	provider      TEXT NOT NULL,
	work_id       TEXT NOT NULL,
	season_number INTEGER NOT NULL,
	name          TEXT NOT NULL,
	episode_count INTEGER NOT NULL,
	cached_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	PRIMARY KEY (provider, work_id, season_number)
);
`

const episodeCacheSchema = `
CREATE TABLE IF NOT EXISTS episode_cache (
	provider       TEXT NOT NULL,
	work_id        TEXT NOT NULL,
	season_number  INTEGER NOT NULL,
	episode_number INTEGER NOT NULL,
	name           TEXT NOT NULL,
	overview       TEXT NOT NULL,
	air_date       TEXT NOT NULL,
	cached_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	PRIMARY KEY (provider, work_id, season_number, episode_number)
);
`

func getCachedSeasons(db *sql.DB, provider, workID string) ([]season, bool) {
	rows, err := db.Query(
		`SELECT season_number, name, episode_count FROM season_cache
		 WHERE provider = ? AND work_id = ? ORDER BY season_number ASC`,
		provider, workID)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	var seasons []season
	for rows.Next() {
		var s season
		if err := rows.Scan(&s.Number, &s.Name, &s.EpisodeCount); err != nil {
			return nil, false
		}
		seasons = append(seasons, s)
	}
	if err := rows.Err(); err != nil || len(seasons) == 0 {
		return nil, false
	}
	return seasons, true
}

func setCachedSeasons(db *sql.DB, provider, workID string, seasons []season) error {
	for _, s := range seasons {
		_, err := db.Exec(
			`INSERT INTO season_cache (provider, work_id, season_number, name, episode_count)
			 VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT(provider, work_id, season_number) DO UPDATE SET
			   name = excluded.name, episode_count = excluded.episode_count`,
			provider, workID, s.Number, s.Name, s.EpisodeCount,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func getCachedEpisodes(db *sql.DB, provider, workID string, seasonNumber int) ([]episode, bool) {
	rows, err := db.Query(
		`SELECT episode_number, name, overview, air_date FROM episode_cache
		 WHERE provider = ? AND work_id = ? AND season_number = ? ORDER BY episode_number ASC`,
		provider, workID, seasonNumber)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	var episodes []episode
	for rows.Next() {
		var e episode
		if err := rows.Scan(&e.Number, &e.Name, &e.Overview, &e.AirDate); err != nil {
			return nil, false
		}
		episodes = append(episodes, e)
	}
	if err := rows.Err(); err != nil || len(episodes) == 0 {
		return nil, false
	}
	return episodes, true
}

func setCachedEpisodes(db *sql.DB, provider, workID string, seasonNumber int, episodes []episode) error {
	for _, e := range episodes {
		_, err := db.Exec(
			`INSERT INTO episode_cache (provider, work_id, season_number, episode_number, name, overview, air_date)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(provider, work_id, season_number, episode_number) DO UPDATE SET
			   name = excluded.name, overview = excluded.overview, air_date = excluded.air_date`,
			provider, workID, seasonNumber, e.Number, e.Name, e.Overview, e.AirDate,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// fetchSeasons/fetchEpisodes: confirmed field names against the real
// API before writing this — seasons come from GET /tv/{id} itself
// (season_number/name/episode_count), episodes from the per-season
// endpoint (episode_number/name/overview/air_date). TV-only, matches
// the Beta 2 scope correction (season/episode belongs to notes, and
// notes are TV-only for this pass).
func fetchSeasons(tvID string) ([]season, error) {
	apiKey := os.Getenv("TMDB_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("TMDB_API_KEY not set")
	}

	u := fmt.Sprintf("https://api.themoviedb.org/3/tv/%s?api_key=%s",
		url.PathEscape(tvID), url.QueryEscape(apiKey))
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned %d for tv/%s", resp.StatusCode, tvID)
	}

	var body struct {
		Seasons []struct {
			SeasonNumber int    `json:"season_number"`
			Name         string `json:"name"`
			EpisodeCount int    `json:"episode_count"`
		} `json:"seasons"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	seasons := make([]season, 0, len(body.Seasons))
	for _, s := range body.Seasons {
		seasons = append(seasons, season{Number: s.SeasonNumber, Name: s.Name, EpisodeCount: s.EpisodeCount})
	}
	return seasons, nil
}

func fetchEpisodes(tvID string, seasonNumber int) ([]episode, error) {
	apiKey := os.Getenv("TMDB_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("TMDB_API_KEY not set")
	}

	u := fmt.Sprintf("https://api.themoviedb.org/3/tv/%s/season/%d?api_key=%s",
		url.PathEscape(tvID), seasonNumber, url.QueryEscape(apiKey))
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB returned %d for tv/%s/season/%d", resp.StatusCode, tvID, seasonNumber)
	}

	var body struct {
		Episodes []struct {
			EpisodeNumber int    `json:"episode_number"`
			Name          string `json:"name"`
			Overview      string `json:"overview"`
			AirDate       string `json:"air_date"`
		} `json:"episodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	episodes := make([]episode, 0, len(body.Episodes))
	for _, e := range body.Episodes {
		episodes = append(episodes, episode{
			Number: e.EpisodeNumber, Name: e.Name, Overview: e.Overview, AirDate: e.AirDate,
		})
	}
	return episodes, nil
}

// displaySeasons/displayEpisodes: cache-first, same fail-open shape as
// displayWork — an empty slice on failure instead of breaking the page.
func displaySeasons(db *sql.DB, provider, workID string) []season {
	if s, ok := getCachedSeasons(db, provider, workID); ok {
		return s
	}
	seasons, err := fetchSeasons(workID)
	if err != nil {
		log.Printf("could not fetch seasons for %s/%s: %v", provider, workID, err)
		return nil
	}
	if err := setCachedSeasons(db, provider, workID, seasons); err != nil {
		log.Printf("failed to cache seasons for %s/%s: %v", provider, workID, err)
	}
	return seasons
}

func displayEpisodes(db *sql.DB, provider, workID string, seasonNumber int) []episode {
	if e, ok := getCachedEpisodes(db, provider, workID, seasonNumber); ok {
		return e
	}
	episodes, err := fetchEpisodes(workID, seasonNumber)
	if err != nil {
		log.Printf("could not fetch episodes for %s/%s season %d: %v", provider, workID, seasonNumber, err)
		return nil
	}
	if err := setCachedEpisodes(db, provider, workID, seasonNumber, episodes); err != nil {
		log.Printf("failed to cache episodes for %s/%s season %d: %v", provider, workID, seasonNumber, err)
	}
	return episodes
}
