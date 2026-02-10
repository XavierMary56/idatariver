package httpapi

import (
	"github.com/gin-gonic/gin"

	"idatariver-finapi/internal/service"
)

func NewRouter(apiSvc *service.APIService, adminSvc *service.AdminService, auth AuthLookup, healthWorstDefault int) *gin.Engine {
	h := &Handler{API: apiSvc, Admin: adminSvc, HealthWorstDefault: healthWorstDefault}
	r := gin.Default()
	h.registerPublic(r)
	h.registerProtected(r, auth)
	h.registerAdmin(r, auth)
	return r
}
