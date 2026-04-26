package videos_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"backend/internal/modules/videos"
)

func TestExtractYouTubeID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		url  string
		want string
		err  bool
	}{
		{"watch", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"watch_no_www", "https://youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"shorts", "https://www.youtube.com/shorts/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"embed", "https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"youtu_be", "https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"with_query", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=42s", "dQw4w9WgXcQ", false},
		{"music", "https://music.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ", false},
		{"invalid_host", "https://vimeo.com/123456", "", true},
		{"empty", "", "", true},
		{"garbled", "not a url", "", true},
		{"missing_v", "https://www.youtube.com/watch", "", true},
		{"too_short", "https://youtu.be/abc", "", true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := videos.ExtractYouTubeID(tc.url)
			if tc.err {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestThumbnailURL(t *testing.T) {
	t.Parallel()
	assert.Equal(t,
		"https://img.youtube.com/vi/abc12345678/hqdefault.jpg",
		videos.ThumbnailURL("abc12345678"),
	)
}

func TestWatchURL(t *testing.T) {
	t.Parallel()
	assert.Equal(t,
		"https://www.youtube.com/watch?v=abc12345678",
		videos.WatchURL("abc12345678"),
	)
}
