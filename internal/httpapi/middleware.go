package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"idatariver-finapi/internal/repo"
	"idatariver-finapi/internal/util"
)

type AuthLookup interface {
	LookupKeyHash(ctx context.Context, keyHash string) (*repo.KeyRecord, error)
}

func APIKeyMiddleware(auth AuthLookup) gin.HandlerFunc {
	return keyAuthMiddleware(auth, "X-API-Key", "invalid api key", false)
}

func AdminKeyMiddleware(auth AuthLookup) gin.HandlerFunc {
	return keyAuthMiddleware(auth, "X-Admin-Key", "invalid admin key", true)
}

func keyAuthMiddleware(auth AuthLookup, headerName, invalidMsg string, requireAdmin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader(headerName))
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing " + headerName})
			return
		}
		hash := util.SHA256Hex(key)
		rec, err := auth.LookupKeyHash(c.Request.Context(), hash)
		if err != nil || rec == nil || !rec.Active || (requireAdmin && !rec.IsAdmin) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": invalidMsg})
			return
		}
		c.Set("tenant_id", rec.TenantID)
		c.Next()
	}
}
