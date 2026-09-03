package server

import (
	"backend/internal/service"

	"github.com/go-kratos/kratos/v2/transport/http"
)

// RegisterAdminLegacyRoutes mounts /api/admin_dhb/* compatibility handlers.
func RegisterAdminLegacyRoutes(srv *http.Server, legacy *service.AdminLegacyService) {
	r := srv.Route("/")
	p := "/api/admin_dhb"

	r.POST(p+"/login", legacy.HandleLogin)
	r.GET(p+"/my_auth_list", legacy.HandleMyAuthList)
	r.GET(p+"/operation_log_list", legacy.HandleOperationLogList)
	r.GET(p+"/all", legacy.HandleAll)
	r.GET(p+"/user_list", legacy.HandleUserList)
	r.GET(p+"/config", legacy.HandleConfig)
	r.POST(p+"/config_update", legacy.HandleConfigUpdate)
	r.GET(p+"/buy_list", legacy.HandleBuyList)
	r.GET(p+"/buy_list_two", legacy.HandleBuyList)
	r.GET(p+"/buy_list_three", legacy.HandleBuyList)
	r.GET(p+"/buy_four_list", legacy.HandleBuyList)
	r.GET(p+"/withdraw_list", legacy.HandleWithdrawList)
	r.POST(p+"/withdraw_pass", legacy.HandleWithdrawPass)
	r.POST(p+"/withdraw_reject", legacy.HandleWithdrawReject)
	r.GET(p+"/exchange_list", legacy.HandleExchangeList)
	r.POST(p+"/exchange_pass", legacy.HandleExchangePass)
	r.POST(p+"/exchange_reject", legacy.HandleExchangeReject)
	r.GET(p+"/transfer_list", legacy.HandleTransferList)
	r.GET(p+"/partner_credit_list", legacy.HandlePartnerCreditList)
	r.GET(p+"/partner_credit_partners", legacy.HandlePartnerCreditPartners)
	r.GET(p+"/reward_list", legacy.HandleRewardList)
	r.GET(p+"/record_list", legacy.HandleRecordList)
	r.GET(p+"/record_list_export", legacy.HandleRecordListExport)
	r.POST(p+"/admin_recharge", legacy.HandleAdminRecharge)
	r.POST(p+"/admin_recharge_win", legacy.HandleAdminRechargeWin)
	r.GET(p+"/good_list", legacy.HandleGoodList)
	r.GET(p+"/good_list_two", legacy.HandleStubGoods)
	r.GET(p+"/good_list_three", legacy.HandleGoodList)
	r.POST(p+"/lock_user_reward", legacy.HandleLockUserReward)
	r.POST(p+"/location_insert", legacy.HandleLocationInsert)
	r.GET(p+"/settlement_list", legacy.HandleSettlementList)
	r.POST(p+"/settlement_trigger", legacy.HandleSettlementTrigger)
	r.POST(p+"/vip_update", legacy.HandleVipUpdate)
	r.POST(p+"/set_zero_account", legacy.HandleSetZeroAccount)
	r.POST(p+"/set_community_subsidy", legacy.HandleSetCommunitySubsidy)
	r.POST(p+"/set_frozen", legacy.HandleSetFrozen)
	r.POST(p+"/set_exchange_enabled", legacy.HandleSetExchangeEnabled)
	r.POST(p+"/set_inviter", legacy.HandleSetInviter)
	r.POST(p+"/change_address", legacy.HandleChangeAddress)
	r.POST(p+"/update_goods", legacy.HandleUpdateGoods)
	r.POST(p+"/update_goods_two", legacy.HandleUpdateGoods)
	r.POST(p+"/update_goods_three", legacy.HandleUpdateGoods)
	r.POST(p+"/upload", legacy.HandleUploadGoods)
	r.POST(p+"/upload_two", legacy.HandleUploadGoods)
	r.POST(p+"/upload_three", legacy.HandleUploadGoods)

	r.GET(p+"/announcement_list", legacy.HandleAnnouncementList)
	r.GET(p+"/announcement_detail", legacy.HandleAnnouncementDetail)
	r.POST(p+"/announcement_detail", legacy.HandleAnnouncementDetail)
	r.POST(p+"/announcement_save", legacy.HandleAnnouncementSave)
	r.POST(p+"/announcement_delete", legacy.HandleAnnouncementDelete)

	// 用户端公告（无需登录）
	r.GET("/v1/announcements", legacy.HandlePublicAnnouncementList)
	r.GET("/v1/announcement/detail", legacy.HandlePublicAnnouncementDetail)

	r.GET(p+"/feedback_list", legacy.HandleFeedbackList)
	r.POST(p+"/feedback_status", legacy.HandleFeedbackStatus)
	r.POST("/v1/feedback", legacy.HandlePublicFeedbackSubmit)
	r.GET("/v1/market/kline", legacy.HandlePublicMarketKline)

	stub := legacy.HandleStubOK
	stubRewards := legacy.HandleStubRewards
	stubLocations := legacy.HandleStubLocations

	r.GET(p+"/sub_money", stub)
	r.POST(p+"/sub_money", stub)
	r.POST(p+"/set_buy_four", stub)
	r.POST(p+"/update_buy_four", stub)
	r.GET(p+"/location_list", stubLocations)
	r.GET(p+"/location_list_2", stubLocations)
	r.GET(p+"/month_recommend", stubRewards)
	r.GET(p+"/user_recommend", legacy.HandleUserRecommend)

	r.GET(p+"/admin_list", stub)
	r.POST(p+"/change_password", stub)
	r.POST(p+"/create_account", stub)
	r.GET(p+"/auth_list", stub)
	r.GET(p+"/user_auth_list", stub)
	r.POST(p+"/auth_create", stub)
	r.POST(p+"/auth_delete", stub)
	r.POST(p+"/amount_four_update", stub)
	r.POST(p+"/add_money_two", stub)
	r.POST(p+"/set_ispay", stub)
	r.POST(p+"/add_money_three", stub)
	r.POST(p+"/set_pass", stub)
	r.POST(p+"/admin_update_location_new_max", stub)
	r.POST(p+"/password_update", stub)
	r.POST(p+"/lock_user", stub)
	r.POST(p+"/admin_recommend_level", stub)
	r.POST(p+"/undo_update", stub)
	r.POST(p+"/level_update", stub)
	r.POST(p+"/vip_delete", stub)
}
