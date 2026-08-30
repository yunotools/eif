package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func staticFallback(staticDir string) gin.HandlerFunc {
	root, _ := filepath.Abs(staticDir)

	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet &&
			c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}

		if strings.HasPrefix(
			c.Request.URL.Path,
			"/api/",
		) {
			c.Status(http.StatusNotFound)
			return
		}

		path := strings.TrimPrefix(
			filepath.Clean("/"+c.Request.URL.Path),
			string(filepath.Separator),
		)

		candidates := []string{
			filepath.Join(root, path),
			filepath.Join(root, path, "index.html"),
		}

		if filepath.Ext(path) == "" && path != "" {
			candidates = append(
				candidates,
				filepath.Join(root, path+".html"),
			)
		}
		if path == "" || path == "." {
			candidates = append([]string{
				filepath.Join(root, "index.html")},
				candidates...,
			)
		}

		for _, candidate := range candidates {
			candidateAbs, err := filepath.Abs(candidate)
			if err != nil || !inRoot(root, candidateAbs) {
				continue
			}
			info, err := os.Stat(candidateAbs)
			if err == nil && !info.IsDir() {
				c.File(candidateAbs)
				return
			}
		}

		index := filepath.Join(root, "index.html")
		if info, err := os.Stat(index); err == nil && !info.IsDir() {
			c.File(index)
			return
		}

		c.Status(http.StatusNotFound)
	}
}

// Đảm bảo: candidate phải nằm trong /app/web/static
// Chống Path traversal
func inRoot(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel != ".." &&
		!strings.HasPrefix(
			rel,
			".."+string(filepath.Separator),
		)
}
