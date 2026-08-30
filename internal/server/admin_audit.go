package server

import (
	"net/http"
	"strings"

	"backend/internal/service"
)

// adminLegacyAuditFilter 记录 /api/admin_dhb 管理后台操作。
func adminLegacyAuditFilter(legacy *service.AdminLegacyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if strings.HasPrefix(path, "/api/admin_dhb/") && path != "/api/admin_dhb/login" {
				legacy.RecordAdminHTTPRequest(r)
			}
			next.ServeHTTP(w, r)
		})
	}
}
