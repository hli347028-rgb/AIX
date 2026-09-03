package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"backend/internal/biz"
	"backend/internal/conf"
	"backend/internal/data"
	jwtpkg "backend/internal/pkg/token"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

var adminActionLabels = map[string]string{
	"/api/admin_dhb/login":                 "登录",
	"/api/admin_dhb/my_auth_list":          "获取权限菜单",
	"/api/admin_dhb/all":                   "查看数据统计",
	"/api/admin_dhb/user_list":             "查看用户列表",
	"/api/admin_dhb/config":                "查看系统配置",
	"/api/admin_dhb/config_update":         "更新系统配置",
	"/api/admin_dhb/buy_list":              "查看认购记录",
	"/api/admin_dhb/buy_list_two":          "查看认购记录",
	"/api/admin_dhb/buy_list_three":        "查看认购记录",
	"/api/admin_dhb/buy_four_list":         "查看认购记录",
	"/api/admin_dhb/withdraw_list":         "查看提现记录",
	"/api/admin_dhb/withdraw_pass":         "提现审核通过",
	"/api/admin_dhb/withdraw_reject":       "提现审核拒绝",
	"/api/admin_dhb/exchange_list":         "查看兑换记录",
	"/api/admin_dhb/exchange_pass":         "兑换审核通过",
	"/api/admin_dhb/exchange_reject":       "兑换审核拒绝",
	"/api/admin_dhb/transfer_list":         "查看划转记录",
	"/api/admin_dhb/partner_credit_list":   "查看交易所划转",
	"/api/admin_dhb/partner_credit_partners": "查看交易所合作方",
	"/api/admin_dhb/reward_list":           "查看奖励记录",
	"/api/admin_dhb/record_list":           "查看流水记录",
	"/api/admin_dhb/record_list_export":    "导出流水记录",
	"/api/admin_dhb/admin_recharge":        "后台USDT充值",
	"/api/admin_dhb/admin_recharge_win":    "后台WIN充值",
	"/api/admin_dhb/settlement_list":       "查看结算列表",
	"/api/admin_dhb/settlement_trigger":    "触发结算",
	"/api/admin_dhb/vip_update":            "更新管理等级",
	"/api/admin_dhb/set_zero_account":      "设置零号账户",
	"/api/admin_dhb/set_community_subsidy": "设置社区补贴",
	"/api/admin_dhb/set_frozen":            "冻结/解冻账户",
	"/api/admin_dhb/set_exchange_enabled":  "开关用户兑换功能",
	"/api/admin_dhb/set_inviter":           "设置上级",
	"/api/admin_dhb/change_address":        "修改用户地址",
	"/api/admin_dhb/announcement_list":     "查看公告列表",
	"/api/admin_dhb/announcement_detail":   "查看公告详情",
	"/api/admin_dhb/announcement_save":     "保存公告",
	"/api/admin_dhb/announcement_delete":   "删除公告",
	"/api/admin_dhb/feedback_list":       "查看问题反馈",
	"/api/admin_dhb/feedback_status":     "更新反馈状态",
	"/api/admin_dhb/operation_log_list":    "查看操作记录",
}

func adminActionLabel(path string) string {
	path = strings.TrimSuffix(strings.TrimSpace(path), "/")
	if label, ok := adminActionLabels[path]; ok {
		return label
	}
	return path
}

func summarizeAdminRequestParams(r *http.Request) string {
	if r == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if q := strings.TrimSpace(r.URL.RawQuery); q != "" {
		parts = append(parts, "query="+q)
	}
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		_ = r.ParseForm()
		if len(r.PostForm) > 0 {
			parts = append(parts, "form="+r.PostForm.Encode())
		}
	}
	out := strings.Join(parts, "; ")
	if len(out) > 2000 {
		return out[:2000]
	}
	return out
}

// shouldRecordAdminHTTPRequest 仅记录变更类操作，查询（GET 等）不写入操作记录。
func shouldRecordAdminHTTPRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(r.Method)) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func clientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return xff
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func tokenFromHTTPRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if t := strings.TrimSpace(r.Header.Get("Access-Token")); t != "" {
		return t
	}
	if t := strings.TrimSpace(r.Header.Get("token")); t != "" {
		return t
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func (s *AdminLegacyService) sessionFromToken(ctx context.Context, tokenString string) (*AdminSession, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, fmt.Errorf("no token")
	}
	addr, err := jwtpkg.Parse(tokenString, s.authCfg.GetJwtSecret())
	if err != nil {
		return nil, err
	}
	if addr == biz.ZeroAddress {
		return &AdminSession{Operator: s.authCfg.GetAdminAccount(), IsMain: true}, nil
	}
	if sub := conf.SubAccountFromJWTAddress(addr); sub != "" {
		if !s.isConfiguredSubAccount(sub) {
			return nil, fmt.Errorf("invalid sub account")
		}
		return &AdminSession{Operator: sub, IsMain: false}, nil
	}
	if ctx != nil {
		user, err := s.admin.RequireAdminUser(ctx, tokenString)
		if err != nil {
			return nil, err
		}
		op := strings.TrimSpace(user.Address)
		if op == "" {
			op = "admin"
		}
		return &AdminSession{Operator: op, IsMain: false}, nil
	}
	return nil, fmt.Errorf("unsupported admin token")
}

var adminSkipAuditPaths = map[string]bool{
	"/api/admin_dhb/my_auth_list": true, // 获取菜单/模块权限，不记操作记录
}

// RecordAdminHTTPRequest 记录管理后台 HTTP 请求（由中间件调用）。
func (s *AdminLegacyService) RecordAdminHTTPRequest(r *http.Request) {
	if r == nil || s == nil || s.data == nil {
		return
	}
	path := strings.TrimSpace(r.URL.Path)
	if !strings.HasPrefix(path, "/api/admin_dhb/") || path == "/api/admin_dhb/login" {
		return
	}
	if adminSkipAuditPaths[path] {
		return
	}
	if !shouldRecordAdminHTTPRequest(r) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	session, err := s.sessionFromToken(ctx, tokenFromHTTPRequest(r))
	if err != nil {
		return
	}
	s.recordAdminOperation(ctx, session, path, r.Method, summarizeAdminRequestParams(r), clientIPFromRequest(r))
}

func (s *AdminLegacyService) recordAdminOperation(ctx context.Context, session *AdminSession, action, method, params, clientIP string) {
	if session == nil || s == nil || s.data == nil {
		return
	}
	operatorType := "sub"
	if session.IsMain {
		operatorType = "main"
	}
	if len(params) > 2000 {
		params = params[:2000]
	}
	po := &data.AdminOperationLogPO{
		Operator:     session.Operator,
		OperatorType: operatorType,
		Action:       action,
		ActionLabel:  adminActionLabel(action),
		Method:       method,
		Params:       params,
		ClientIP:     clientIP,
	}
	if err := s.data.DB().WithContext(ctx).Create(po).Error; err != nil {
		return
	}
}

// HandleOperationLogList 主账户查看操作记录。
func (s *AdminLegacyService) HandleOperationLogList(ctx khttp.Context) error {
	if _, err := s.requireMainAdmin(ctx); err != nil {
		return err
	}
	q := ctx.Request().URL.Query()
	page, pageSize, offset := parsePage(q)
	operatorFilter := strings.TrimSpace(q.Get("operator"))
	actionFilter := strings.TrimSpace(q.Get("action"))

	db := s.data.DB().WithContext(ctx).Model(&data.AdminOperationLogPO{})
	if operatorFilter != "" {
		db = db.Where("operator LIKE ?", "%"+operatorFilter+"%")
	}
	if actionFilter != "" {
		db = db.Where("action_label LIKE ? OR action LIKE ?", "%"+actionFilter+"%", "%"+actionFilter+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}
	var rows []data.AdminOperationLogPO
	if err := db.Order("id desc").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return err
	}
	list := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		list = append(list, map[string]interface{}{
			"id":            row.ID,
			"operator":      row.Operator,
			"operator_type": row.OperatorType,
			"action":        row.Action,
			"action_label":  row.ActionLabel,
			"method":        row.Method,
			"params":        row.Params,
			"client_ip":     row.ClientIP,
			"created_at":    formatLegacyTime(row.CreatedTime),
		})
	}
	return ctx.Result(200, map[string]interface{}{
		"list":  list,
		"count": total,
		"page":  page,
	})
}

func (s *AdminLegacyService) logAdminLogin(ctx context.Context, session *AdminSession, r *http.Request) {
	if session == nil {
		return
	}
	params := ""
	if r != nil {
		params = summarizeAdminRequestParams(r)
	}
	s.recordAdminOperation(ctx, session, "/api/admin_dhb/login", http.MethodPost, params, clientIPFromRequest(r))
}
