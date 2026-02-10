package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"idatariver-finapi/internal/service"
)

type Handler struct {
	API                *service.APIService
	Admin              *service.AdminService
	HealthWorstDefault int
}

func (h *Handler) registerPublic(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

func (h *Handler) registerProtected(r *gin.Engine, auth AuthLookup) {
	protected := r.Group("/api/v1", APIKeyMiddleware(auth))

	protected.GET("/meta/fields", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": service.AvailableFieldMeta()})
	})

	protected.GET("/meta/symbols", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		types := splitCSV(c.Query("types"))
		onlyActive := parseBoolParam(c.Query("active"), true)
		data, err := h.API.ListSymbols(c.Request.Context(), tenantID, onlyActive, types)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	})

	protected.GET("/market/compare/latest", func(c *gin.Context) {
		symbols := splitCSV(c.Query("symbols"))
		fields := splitCSV(c.Query("fields"))
		includeClose := parseBoolParam(c.Query("include_close"), true)
		sortBy := c.Query("sort_by")
		order := c.Query("order")
		data, err := h.API.CompareLatest(c.Request.Context(), symbols, fields, includeClose, sortBy, order)
		if err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	})

	protected.GET("/health/data", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		types := splitCSV(c.Query("types"))
		worstN := parsePositiveInt(c.Query("worst"), h.HealthWorstDefault)
		includeItems := parseBoolParam(c.Query("include_items"), false)
		rep, err := h.API.DataHealth(c.Request.Context(), tenantID, types, worstN, includeItems)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, rep)
	})
}

func (h *Handler) registerAdmin(r *gin.Engine, auth AuthLookup) {
	admin := r.Group("/api/v1/admin", AdminKeyMiddleware(auth))

	admin.GET("/sla", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		data, err := h.Admin.ListInstrumentSLA(c.Request.Context(), tenantID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
	})

	admin.POST("/sla", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		var req service.SetSLARequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "invalid json")
			return
		}
		if err := h.Admin.SetInstrumentSLA(c.Request.Context(), tenantID, req.Symbol, req.SLAHours); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	admin.DELETE("/sla", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		symbol := strings.TrimSpace(c.Query("symbol"))
		if symbol == "" {
			respondError(c, http.StatusBadRequest, "missing symbol")
			return
		}
		if err := h.Admin.DeleteInstrumentSLA(c.Request.Context(), tenantID, symbol); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	admin.POST("/sla/batch", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		var req service.SetSLABatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "invalid json")
			return
		}
		res, err := h.Admin.SetInstrumentSLABatch(c.Request.Context(), tenantID, req.Items)
		if err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, res)
	})

	admin.GET("/sla.csv", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		csvText, err := h.Admin.ExportSLAAsCSV(c.Request.Context(), tenantID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=\"instrument_sla.csv\"")
		c.String(http.StatusOK, csvText)
	})

	admin.POST("/sla.csv", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		b, err := readAll(c)
		if err != nil {
			respondError(c, http.StatusBadRequest, "failed to read body")
			return
		}
		res, err := h.Admin.ImportSLAFromCSV(c.Request.Context(), tenantID, string(b))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "updated": res.Updated, "failed": res.Failed})
			return
		}
		c.JSON(http.StatusOK, res)
	})

	admin.GET("/health/snooze", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		resp, err := h.Admin.GetSnoozeStatus(c.Request.Context(), tenantID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, resp)
	})
	admin.POST("/health/snooze", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		var req service.SnoozeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "invalid json")
			return
		}
		resp, err := h.Admin.Snooze(c.Request.Context(), tenantID, req.Minutes, req.Reason)
		if err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, resp)
	})
	admin.DELETE("/health/snooze", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		resp, err := h.Admin.CancelSnooze(c.Request.Context(), tenantID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, resp)
	})

	admin.GET("/notify", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		cfg, err := h.Admin.GetNotifyConfig(c.Request.Context(), tenantID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, cfg)
	})
	admin.PUT("/notify", func(c *gin.Context) {
		tenantID := tenantIDFromCtx(c)
		var dto service.NotifyConfigDTO
		if err := c.ShouldBindJSON(&dto); err != nil {
			respondError(c, http.StatusBadRequest, "invalid json")
			return
		}
		if err := h.Admin.UpdateNotifyConfig(c.Request.Context(), tenantID, dto); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}
