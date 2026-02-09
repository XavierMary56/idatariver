package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"idatariver-finapi/internal/repo"
	"idatariver-finapi/internal/service"
	"idatariver-finapi/internal/util"
)

func main() {
	dsn := util.Getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/idatariver?sslmode=disable")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil { panic(err) }
	defer pool.Close()

	authRepo := repo.NewAuthRepo(pool)
	healthRepo := repo.NewHealthRepo(pool)
	notifyRepo := repo.NewNotifyRepo(pool)

	apiSvc := &service.APIService{DB: pool}
	adminSvc := &service.AdminService{API: apiSvc, HealthRepo: healthRepo, NotifyRepo: notifyRepo}

	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	protected := r.Group("/api/v1", apiKeyMiddleware(authRepo))
	{
		protected.GET("/meta/fields", func(c *gin.Context) {
			c.JSON(200, gin.H{"data": service.AvailableFieldMeta()})
		})

		protected.GET("/meta/symbols", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			types := splitCSV(c.Query("types"))
			onlyActive := strings.ToLower(strings.TrimSpace(c.DefaultQuery("active", "true"))) != "false"
			data, err := apiSvc.ListSymbols(c.Request.Context(), tenantID, onlyActive, types)
			if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
			c.JSON(200, gin.H{"data": data})
		})

		protected.GET("/market/compare/latest", func(c *gin.Context) {
			symbols := splitCSV(c.Query("symbols"))
			fields := splitCSV(c.Query("fields"))
			includeClose := strings.ToLower(strings.TrimSpace(c.DefaultQuery("include_close", "true"))) != "false"
			sortBy := c.Query("sort_by")
			order := c.Query("order")
			data, err := apiSvc.CompareLatest(c.Request.Context(), symbols, fields, includeClose, sortBy, order)
			if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			c.JSON(200, gin.H{"data": data})
		})

		protected.GET("/health/data", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			types := splitCSV(c.Query("types"))
			worstN := util.GetenvInt("HEALTH_WORST_DEFAULT", 10)
			if v := strings.TrimSpace(c.Query("worst")); v != "" {
				if n := util.AtoiSafe(v); n > 0 { worstN = n }
			}
			includeItems := strings.ToLower(strings.TrimSpace(c.DefaultQuery("include_items", "false"))) == "true"
			rep, err := apiSvc.DataHealth(c.Request.Context(), tenantID, types, worstN, includeItems)
			if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
			c.JSON(200, rep)
		})
	}

	admin := r.Group("/api/v1/admin", adminKeyMiddleware(authRepo))
	{
		admin.GET("/sla", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			data, err := adminSvc.ListInstrumentSLA(c.Request.Context(), tenantID)
			if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
			c.JSON(200, gin.H{"data": data})
		})

		admin.POST("/sla", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			var req service.SetSLARequest
			if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid json"}); return }
			if err := adminSvc.SetInstrumentSLA(c.Request.Context(), tenantID, req.Symbol, req.SLAHours); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			c.JSON(200, gin.H{"ok": true})
		})

		admin.DELETE("/sla", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			symbol := strings.TrimSpace(c.Query("symbol"))
			if symbol == "" { c.JSON(400, gin.H{"error": "missing symbol"}); return }
			if err := adminSvc.DeleteInstrumentSLA(c.Request.Context(), tenantID, symbol); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			c.JSON(200, gin.H{"ok": true})
		})

		admin.POST("/sla/batch", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			var req service.SetSLABatchRequest
			if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid json"}); return }
			res, err := adminSvc.SetInstrumentSLABatch(c.Request.Context(), tenantID, req.Items)
			if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			c.JSON(200, res)
		})

		admin.GET("/sla.csv", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			csvText, err := adminSvc.ExportSLAAsCSV(c.Request.Context(), tenantID)
			if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
			c.Header("Content-Type", "text/csv; charset=utf-8")
			c.Header("Content-Disposition", "attachment; filename=\"instrument_sla.csv\"")
			c.String(200, csvText)
		})

		admin.POST("/sla.csv", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			b, err := ioReadAll(c)
			if err != nil { c.JSON(400, gin.H{"error": "failed to read body"}); return }
			res, err := adminSvc.ImportSLAFromCSV(c.Request.Context(), tenantID, string(b))
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error(), "updated": res.Updated, "failed": res.Failed})
				return
			}
			c.JSON(200, res)
		})

		// Snooze
		admin.GET("/health/snooze", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			resp, err := adminSvc.GetSnoozeStatus(c.Request.Context(), tenantID)
			if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
			c.JSON(200, resp)
		})
		admin.POST("/health/snooze", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			var req service.SnoozeRequest
			if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": "invalid json"}); return }
			resp, err := adminSvc.Snooze(c.Request.Context(), tenantID, req.Minutes, req.Reason)
			if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			c.JSON(200, resp)
		})
		admin.DELETE("/health/snooze", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			resp, err := adminSvc.CancelSnooze(c.Request.Context(), tenantID)
			if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
			c.JSON(200, resp)
		})

		// Notify config
		admin.GET("/notify", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			cfg, err := adminSvc.GetNotifyConfig(c.Request.Context(), tenantID)
			if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
			c.JSON(200, cfg)
		})
		admin.PUT("/notify", func(c *gin.Context) {
			tenantID := tenantIDFromCtx(c)
			var dto service.NotifyConfigDTO
			if err := c.ShouldBindJSON(&dto); err != nil { c.JSON(400, gin.H{"error": "invalid json"}); return }
			if err := adminSvc.UpdateNotifyConfig(c.Request.Context(), tenantID, dto); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			c.JSON(200, gin.H{"ok": true})
		})
	}

	addr := util.Getenv("API_ADDR", ":8080")
	_ = os.Setenv("GIN_MODE", util.Getenv("GIN_MODE", "release"))
	if err := r.Run(addr); err != nil { panic(err) }
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" { return []string{} }
	parts := strings.Split(s, ",")
	out := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" { out = append(out, p) }
	}
	return out
}

func tenantIDFromCtx(c *gin.Context) int64 {
	v, ok := c.Get("tenant_id")
	if !ok { return 0 }
	id, _ := v.(int64)
	return id
}

func apiKeyMiddleware(auth *repo.AuthRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-API-Key"})
			return
		}
		hash := util.SHA256Hex(key)
		rec, err := auth.LookupKeyHash(c.Request.Context(), hash)
		if err != nil || rec == nil || !rec.Active {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}
		c.Set("tenant_id", rec.TenantID)
		c.Next()
	}
}

func adminKeyMiddleware(auth *repo.AuthRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader("X-Admin-Key"))
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-Admin-Key"})
			return
		}
		hash := util.SHA256Hex(key)
		rec, err := auth.LookupKeyHash(c.Request.Context(), hash)
		if err != nil || rec == nil || !rec.Active || !rec.IsAdmin {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid admin key"})
			return
		}
		c.Set("tenant_id", rec.TenantID)
		c.Next()
	}
}

func ioReadAll(c *gin.Context) ([]byte, error) {
	return io.ReadAll(c.Request.Body)
}

