package httpapi

import (
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func tenantIDFromCtx(c *gin.Context) int64 {
	v, ok := c.Get("tenant_id")
	if !ok {
		return 0
	}
	id, _ := v.(int64)
	return id
}

func readAll(c *gin.Context) ([]byte, error) {
	return io.ReadAll(c.Request.Body)
}

func respondError(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

func parseBoolParam(raw string, def bool) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return def
	}
	if raw == "true" {
		return true
	}
	if raw == "false" {
		return false
	}
	return def
}

func parsePositiveInt(raw string, def int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
