# SortHero

A powerful media file organizer that automatically renames and sorts your movies and TV shows using TMDB and TVDB data.
This is still in beta but works really well, i will add some improvements

## Features

- Automatic media detection and parsing
- Movie info from TMDB API
- TV show info from TVDB API
- Supports multiple file operations (move/copy/symlink)
- Dry-run mode for safe testing
- Debug mode for troubleshooting
- Configurable source and destination directories
- Supports common video formats (MP4, MKV, AVI, etc.)

## Installation

```bash
go get github.com/yourusername/sorthero
```

## Configuration

Create a `config.json` file:

```json
{
    "tmdb_api_key": "your_tmdb_key_here",
    "tvdb_api_key": "your_tvdb_key_here"
}
```

Get your API keys from:
- TMDB: https://www.themoviedb.org/settings/api
- TVDB: https://thetvdb.com/api-key

## Usage

Basic usage:
```bash
sorthero -source "/path/to/media" -dest "/path/to/library"
```

Available options:
```bash
  -config string
        Path to config file (default "config.json")
  -debug
        Enable debug logging
  -dest string
        Destination directory (default ".")
  -dry-run
        Show what would be done without making changes
  -op string
        Operation: move, copy, or symlink (default "move")
  -source string
        Source directory (default ".")
```

### Examples

Test run with debug info:
```bash
sorthero -source "/downloads" -dest "/media" -debug -dry-run
```

Copy files instead of moving:
```bash
sorthero -source "/downloads" -dest "/media" -op copy
```

Create symlinks:
```bash
sorthero -source "/downloads" -dest "/media" -op symlink
```

## File Naming

### Movies
Input: `Movie.Name.2020.1080p.mkv`
Output: `Movie Name (2020) [1080p].mkv`

### TV Shows
Input: `Show.Name.S01E02.720p.mp4`
Output: `Show Name S01E02 [720p].mp4`

## License

MIT

## Contributing

Pull requests welcome! Please read our contributing guidelines first.
