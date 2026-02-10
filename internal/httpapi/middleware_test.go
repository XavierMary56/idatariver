package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"idatariver-finapi/internal/repo"
)

type fakeAuth struct {
	rec *repo.KeyRecord
	err error
}

func (f fakeAuth) LookupKeyHash(ctx context.Context, keyHash string) (*repo.KeyRecord, error) {
	return f.rec, f.err
}

func TestAPIKeyMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/x", APIKeyMiddleware(fakeAuth{rec: &repo.KeyRecord{TenantID: 7, Active: true}}), func(c *gin.Context) {
		if tenantIDFromCtx(c) != 7 {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-API-Key", "devkey")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAPIKeyMiddleware_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/x", APIKeyMiddleware(fakeAuth{rec: &repo.KeyRecord{TenantID: 7, Active: false}}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-API-Key", "devkey")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAdminKeyMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/x", AdminKeyMiddleware(fakeAuth{rec: &repo.KeyRecord{TenantID: 7, Active: true, IsAdmin: true}}), func(c *gin.Context) {
		if tenantIDFromCtx(c) != 7 {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Admin-Key", "adminkey")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}
