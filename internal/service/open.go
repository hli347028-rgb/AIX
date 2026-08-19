package service

import (
	"net/http"
	"strconv"
	"strings"

	"backend/internal/biz"
	"backend/internal/conf"
	authmw "backend/internal/middleware"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// OpenService exposes third-party Open API endpoints (API Key auth).
type OpenService struct {
	walletRepo biz.WalletRepo
	authCfg    *conf.AuthConfig
	log        *log.Helper
}

func NewOpenService(walletRepo biz.WalletRepo, authCfg *conf.AuthConfig, logger log.Logger) *OpenService {
	return &OpenService{
		walletRepo: walletRepo,
		authCfg:    authCfg,
		log:        log.NewHelper(logger),
	}
}

// RegisterOpenRoutes mounts /v1/open/* routes.
func RegisterOpenRoutes(srv *khttp.Server, open *OpenService) {
	r := srv.Route("/")
	r.GET("/v1/open/subscribe-orders", open.HandleSubscribeOrders)
}

func (s *OpenService) requireAPIKey(ctx khttp.Context) (keyHint string, err error) {
	req := ctx.Request()
	key := strings.TrimSpace(req.Header.Get("X-API-Key"))
	if key == "" {
		key = authmw.ParseBearer(req.Header.Get("Authorization"))
	}
	if key == "" {
		key = strings.TrimSpace(req.URL.Query().Get("api_key"))
	}
	if s.authCfg == nil || !s.authCfg.IsOpenAPIKey(key) {
		return apiKeyHint(key), errors.Unauthorized("INVALID_API_KEY", "invalid or missing api key")
	}
	return apiKeyHint(key), nil
}

// apiKeyHint returns a non-secret fingerprint for logs (never full key).
func apiKeyHint(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "(empty)"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "…" + key[len(key)-4:]
}

func clientIP(req *http.Request) string {
	if req == nil {
		return ""
	}
	if xff := strings.TrimSpace(req.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := strings.TrimSpace(req.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host := req.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		return host[:i]
	}
	return host
}

// HandleSubscribeOrders returns all users' subscribe orders for third parties.
// Fields: order_id, address, principal, points, created_time.
func (s *OpenService) HandleSubscribeOrders(ctx khttp.Context) error {
	req := ctx.Request()
	ip := clientIP(req)
	ua := ""
	if req != nil {
		ua = req.UserAgent()
	}

	keyHint, err := s.requireAPIKey(ctx)
	if err != nil {
		s.log.Warnf("openapi subscribe-orders denied ip=%s key=%s ua=%q", ip, keyHint, ua)
		return err
	}

	q := ctx.Request().URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset := (page - 1) * pageSize
	orders, total, err := s.walletRepo.ListSubscribeOrdersPaged(ctx, offset, pageSize)
	if err != nil {
		s.log.Errorf("openapi subscribe-orders failed ip=%s key=%s page=%d pageSize=%d err=%v", ip, keyHint, page, pageSize, err)
		return err
	}
	items := make([]map[string]any, 0, len(orders))
	for _, o := range orders {
		if o == nil || o.Order == nil {
			continue
		}
		pts := o.Order.Points
		if pts == "" {
			pts = o.Order.Principal
		}
		items = append(items, map[string]any{
			"order_id":     o.Order.ID,
			"address":      o.UserAddress,
			"principal":    o.Order.Principal,
			"points":       pts,
			"created_time": o.Order.CreatedTime.Unix(),
		})
	}
	s.log.Infof("openapi subscribe-orders ok ip=%s key=%s page=%d pageSize=%d returned=%d total=%d",
		ip, keyHint, page, pageSize, len(items), total)
	return ctx.JSON(http.StatusOK, map[string]any{
		"orders":   items,
		"count":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
