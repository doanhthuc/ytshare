package videos

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ErrInvalidYouTubeURL is returned when we cannot extract a video id.
var ErrInvalidYouTubeURL = errors.New("videos: invalid youtube URL")

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// ExtractYouTubeID parses a YouTube URL (watch?v=, youtu.be/, embed/, shorts/)
// and returns the canonical 11-character video id.
func ExtractYouTubeID(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrInvalidYouTubeURL
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidYouTubeURL, err.Error())
	}

	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	switch host {
	case "youtu.be":
		id := strings.Trim(u.Path, "/")
		if idPattern.MatchString(id) {
			return id, nil
		}
	case "youtube.com", "m.youtube.com", "music.youtube.com":
		if v := u.Query().Get("v"); idPattern.MatchString(v) {
			return v, nil
		}
		// /embed/<id>, /shorts/<id>, /v/<id>
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 {
			id := parts[len(parts)-1]
			if idPattern.MatchString(id) {
				return id, nil
			}
		}
	}
	return "", ErrInvalidYouTubeURL
}

// ThumbnailURL returns the standard YouTube thumbnail URL for a video id.
func ThumbnailURL(youtubeID string) string {
	return fmt.Sprintf("https://img.youtube.com/vi/%s/hqdefault.jpg", youtubeID)
}

// WatchURL returns the canonical watch URL for a video id.
func WatchURL(youtubeID string) string {
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", youtubeID)
}
