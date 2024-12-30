package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "time"
)

const (
    TMDB_API_URL = "https://api.themoviedb.org/3"
    TVDB_API_URL = "https://api.thetvdb.com/api"
)

type Config struct {
    TMDBApiKey string
    TVDBApiKey string
    TVDBToken  string
}

type TMDBSearchResult struct {
    Results []struct {
        ID          int    `json:"id"`
        Title       string `json:"title"`
        ReleaseDate string `json:"release_date"`
    } `json:"results"`
}

type TVDBSearchResult struct {
    Data []struct {
        ID           int    `json:"id"`
        SeriesName   string `json:"seriesName"`
        FirstAired   string `json:"firstAired"`
        Status       string `json:"status"`
        Network      string `json:"network"`
    } `json:"data"`
}

type MediaFile struct {
    Path        string
    Type        string
    Title       string
    Year        string
    Season      string
    Episode     string
    Quality     string
    Ext         string
    TargetPath  string
}

var (
    movieRegex = regexp.MustCompile(`(?i)^(.+?)[\.\s]+(19\d{2}|20\d{2})[\.\s]+(720p|1080p|2160p|BRRip|BluRay|WEBRip|HDRip)?`)
    tvRegex    = regexp.MustCompile(`(?i)^(.+?)[\.\s]+[Ss](\d{1,2})[Ee](\d{1,2})[\.\s]*(720p|1080p|2160p|BRRip|BluRay|WEBRip|HDRip)?`)
    config     Config
    client     = &http.Client{Timeout: 10 * time.Second}
    debug      *bool
)

func debugLog(format string, v ...interface{}) {
    if *debug {
        fmt.Printf("[DEBUG] "+format+"\n", v...)
    }
}

func loadConfig(path string) error {
    debugLog("Loading config from: %s", path)
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }

    var cfg struct {
        TMDBApiKey string `json:"tmdb_api_key"`
        TVDBApiKey string `json:"tvdb_api_key"`
    }

    if err := json.Unmarshal(data, &cfg); err != nil {
        return err
    }

    config.TMDBApiKey = cfg.TMDBApiKey
    config.TVDBApiKey = cfg.TVDBApiKey
    debugLog("Config loaded successfully")
    return nil
}

func getTVDBToken() error {
    debugLog("Getting TVDB token")
    data := map[string]string{"apikey": config.TVDBApiKey}
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", TVDB_API_URL+"/login", strings.NewReader(string(jsonData)))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var result struct {
        Token string `json:"token"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return err
    }

    config.TVDBToken = result.Token
    debugLog("TVDB token obtained successfully")
    return nil
}

func searchTMDB(title string) (*TMDBSearchResult, error) {
    query := url.QueryEscape(title)
    endpoint := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s", TMDB_API_URL, config.TMDBApiKey, query)
    debugLog("Searching TMDB for: %s", title)
    
    resp, err := client.Get(endpoint)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    debugLog("TMDB response status: %d", resp.StatusCode)
    var result TMDBSearchResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    if len(result.Results) > 0 {
        debugLog("Found %d results on TMDB", len(result.Results))
    } else {
        debugLog("No results found on TMDB")
    }

    return &result, nil
}

func searchTVDB(title string) (*TVDBSearchResult, error) {
    query := url.QueryEscape(title)
    debugLog("Searching TVDB for: %s", title)
    
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/search/series?name=%s", TVDB_API_URL, query), nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+config.TVDBToken)
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    debugLog("TVDB response status: %d", resp.StatusCode)
    var result TVDBSearchResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    if len(result.Data) > 0 {
        debugLog("Found %d results on TVDB", len(result.Data))
    } else {
        debugLog("No results found on TVDB")
    }

    return &result, nil
}

func parseFile(path string, destDir string) (*MediaFile, error) {
    debugLog("Parsing file: %s", path)
    base := filepath.Base(path)
    ext := filepath.Ext(base)
    name := strings.TrimSuffix(base, ext)

    if matches := tvRegex.FindStringSubmatch(name); matches != nil {
        debugLog("Matched TV show pattern")
        mf := &MediaFile{
            Path:    path,
            Type:    "tv",
            Title:   cleanTitle(matches[1]),
            Season:  matches[2],
            Episode: matches[3],
            Quality: matches[4],
            Ext:     ext,
        }

        if result, err := searchTVDB(mf.Title); err == nil && len(result.Data) > 0 {
            mf.Title = result.Data[0].SeriesName
            if result.Data[0].FirstAired != "" {
                mf.Year = result.Data[0].FirstAired[:4]
            }
            debugLog("Updated TV show info from TVDB: %s (%s)", mf.Title, mf.Year)
        }

        showDir := filepath.Join(destDir, "TV Shows", mf.Title)
        seasonDir := filepath.Join(showDir, fmt.Sprintf("Season %s", mf.Season))
        mf.TargetPath = filepath.Join(seasonDir, mf.NewName())
        debugLog("Target path: %s", mf.TargetPath)
        
        return mf, nil
    }

    if matches := movieRegex.FindStringSubmatch(name); matches != nil {
        debugLog("Matched movie pattern")
        mf := &MediaFile{
            Path:    path,
            Type:    "movie",
            Title:   cleanTitle(matches[1]),
            Year:    matches[2],
            Quality: matches[3],
            Ext:     ext,
        }

        if result, err := searchTMDB(mf.Title); err == nil && len(result.Results) > 0 {
            mf.Title = result.Results[0].Title
            if result.Results[0].ReleaseDate != "" {
                mf.Year = result.Results[0].ReleaseDate[:4]
            }
            debugLog("Updated movie info from TMDB: %s (%s)", mf.Title, mf.Year)
        }

        movieDir := filepath.Join(destDir, "Movies")
        mf.TargetPath = filepath.Join(movieDir, mf.NewName())
        debugLog("Target path: %s", mf.TargetPath)
        
        return mf, nil
    }

    return nil, fmt.Errorf("unable to parse filename: %s", name)
}

func cleanTitle(title string) string {
    title = strings.ReplaceAll(title, ".", " ")
    title = strings.Title(strings.ToLower(title))
    return strings.TrimSpace(title)
}

func (mf *MediaFile) NewName() string {
    var newName string
    if mf.Type == "movie" {
        if mf.Quality != "" {
            newName = fmt.Sprintf("%s (%s) [%s]%s", mf.Title, mf.Year, mf.Quality, mf.Ext)
        } else {
            newName = fmt.Sprintf("%s (%s)%s", mf.Title, mf.Year, mf.Ext)
        }
    } else {
        season := fmt.Sprintf("S%02s", mf.Season)
        episode := fmt.Sprintf("E%02s", mf.Episode)
        if mf.Quality != "" {
            newName = fmt.Sprintf("%s %s%s [%s]%s", mf.Title, season, episode, mf.Quality, mf.Ext)
        } else {
            newName = fmt.Sprintf("%s %s%s%s", mf.Title, season, episode, mf.Ext)
        }
    }
    debugLog("New filename: %s", newName)
    return newName
}

func processFile(mf *MediaFile, operation string, dryRun bool) error {
    debugLog("Processing file with operation: %s", operation)
    targetDir := filepath.Dir(mf.TargetPath)
    
    if !dryRun {
        if err := os.MkdirAll(targetDir, 0755); err != nil {
            return fmt.Errorf("error creating directory %s: %v", targetDir, err)
        }
    }

    switch operation {
    case "copy":
        if dryRun {
            fmt.Printf("Would copy: %s -> %s\n", mf.Path, mf.TargetPath)
            return nil
        }
        debugLog("Copying file")
        return copyFile(mf.Path, mf.TargetPath)
    
    case "move":
        if dryRun {
            fmt.Printf("Would move: %s -> %s\n", mf.Path, mf.TargetPath)
            return nil
        }
        debugLog("Moving file")
        return os.Rename(mf.Path, mf.TargetPath)
    
    case "symlink":
        if dryRun {
            fmt.Printf("Would symlink: %s -> %s\n", mf.Path, mf.TargetPath)
            return nil
        }
        debugLog("Creating symlink")
        return os.Symlink(mf.Path, mf.TargetPath)
    
    default:
        return fmt.Errorf("unknown operation: %s", operation)
    }
}

func copyFile(src, dst string) error {
    debugLog("Copying from %s to %s", src, dst)
    source, err := os.Open(src)
    if err != nil {
        return err
    }
    defer source.Close()

    destination, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destination.Close()

    _, err = io.Copy(destination, source)
    return err
}

func isVideoFile(ext string) bool {
    videoExts := map[string]bool{
        ".mp4":  true,
        ".mkv":  true,
        ".avi":  true,
        ".mov":  true,
        ".wmv":  true,
        ".m4v":  true,
        ".flv":  true,
    }
    return videoExts[ext]
}

func main() {
    var (
        sourceDir  = flag.String("source", ".", "Source directory")
        destDir    = flag.String("dest", ".", "Destination directory")
        operation  = flag.String("op", "move", "Operation: move, copy, or symlink")
        dryRun     = flag.Bool("dry-run", false, "Show what would be done without making changes")
        configPath = flag.String("config", "config.json", "Path to config file")
        debugFlag  = flag.Bool("debug", false, "Enable debug logging")
    )
    
    flag.Parse()
    debug = debugFlag

    if err := loadConfig(*configPath); err != nil {
        fmt.Printf("Error loading config: %v\n", err)
        os.Exit(1)
    }

    if err := getTVDBToken(); err != nil {
        fmt.Printf("Error getting TVDB token: %v\n", err)
        os.Exit(1)
    }

    debugLog("Starting file processing")
    debugLog("Source directory: %s", *sourceDir)
    debugLog("Destination directory: %s", *destDir)
    debugLog("Operation: %s", *operation)
    debugLog("Dry run: %v", *dryRun)

    err := filepath.Walk(*sourceDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if info.IsDir() {
            return nil
        }

        ext := strings.ToLower(filepath.Ext(path))
        if !isVideoFile(ext) {
            debugLog("Skipping non-video file: %s", path)
            return nil
        }

        mediaFile, err := parseFile(path, *destDir)
        if err != nil {
            fmt.Printf("Warning: %v\n", err)
            return nil
        }

        if err := processFile(mediaFile, *operation, *dryRun); err != nil {
            fmt.Printf("Error processing %s: %v\n", path, err)
        }

        return nil
    })

    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    
    debugLog("Processing completed")
}
