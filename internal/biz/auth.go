package biz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	v1 "backend/api/auth/v1"
	"backend/internal/conf"
	"backend/internal/pkg/eth"
	"backend/internal/pkg/token"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/shopspring/decimal"
)

type AuthUsecase struct {
	userRepo      UserRepo
	challengeRepo ChallengeRepo
	auth          *conf.AuthConfig
	log           *log.Helper
}

func NewAuthUsecase(userRepo UserRepo, challengeRepo ChallengeRepo, auth *conf.AuthConfig, logger log.Logger) *AuthUsecase {
	return &AuthUsecase{
		userRepo:      userRepo,
		challengeRepo: challengeRepo,
		auth:          auth,
		log:           log.NewHelper(logger),
	}
}

func (uc *AuthUsecase) challengeTTL() time.Duration {
	return uc.auth.ChallengeDuration()
}

func (uc *AuthUsecase) jwtSecret() string {
	return uc.auth.GetJwtSecret()
}

func (uc *AuthUsecase) isBootstrapAddress(address string) bool {
	for _, item := range uc.auth.GetBootstrapAddresses() {
		normalized, err := eth.NormalizeAddress(item)
		if err == nil && normalized == address {
			return true
		}
	}
	return false
}

func (uc *AuthUsecase) GetChallenge(ctx context.Context, address string) (*Challenge, error) {
	normalized, err := eth.NormalizeAddress(address)
	if err != nil {
		return nil, errors.BadRequest(v1.ErrorReason_INVALID_ADDRESS.String(), "无效的钱包地址")
	}
	nonce, err := randomNonce(16)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	expireAt := now.Add(uc.challengeTTL())
	message := fmt.Sprintf(
		"Welcome to AIX Web3 login\nAddress: %s\nNonce: %s\nTimestamp: %d",
		normalized, nonce, now.Unix(),
	)
	challenge := &Challenge{Address: normalized, Message: message, ExpireAt: expireAt}
	if err := uc.challengeRepo.Save(ctx, challenge); err != nil {
		return nil, err
	}
	return challenge, nil
}

func (uc *AuthUsecase) Login(ctx context.Context, address, signature, inviteCode string) (*User, string, time.Time, bool, error) {
	normalized, err := eth.NormalizeAddress(address)
	if err != nil {
		return nil, "", time.Time{}, false, errors.BadRequest(v1.ErrorReason_INVALID_ADDRESS.String(), "无效的钱包地址")
	}
	isZeroAdminLogin := normalized == ZeroAddress
	if !isZeroAdminLogin {
		challenge, err := uc.challengeRepo.Get(ctx, normalized)
		if err != nil {
			return nil, "", time.Time{}, false, errors.BadRequest(v1.ErrorReason_CHALLENGE_NOT_FOUND.String(), "请先获取签名挑战")
		}
		if challenge == nil {
			return nil, "", time.Time{}, false, errors.BadRequest(v1.ErrorReason_CHALLENGE_NOT_FOUND.String(), "签名挑战不存在或已过期，请重新获取")
		}
		if time.Now().After(challenge.ExpireAt) {
			_ = uc.challengeRepo.Delete(ctx, normalized)
			return nil, "", time.Time{}, false, errors.BadRequest(v1.ErrorReason_CHALLENGE_EXPIRED.String(), "签名挑战已过期，请重新获取")
		}
		if err := eth.VerifyPersonalSign(challenge.Message, signature, normalized); err != nil {
			return nil, "", time.Time{}, false, errors.Unauthorized(v1.ErrorReason_INVALID_SIGNATURE.String(), "签名校验失败")
		}
	}

	existing, err := uc.userRepo.FindByAddress(ctx, normalized)
	if err != nil {
		return nil, "", time.Time{}, false, err
	}
	isNewUser := existing == nil
	var user *User
	if isNewUser {
		user, err = uc.registerNewUser(ctx, normalized, inviteCode)
		if err != nil {
			return nil, "", time.Time{}, false, err
		}
	} else {
		user = existing
		if user.IsFrozen {
			return nil, "", time.Time{}, false, errors.Forbidden(AccountFrozenReason, AccountFrozenMessage)
		}
	}
	if !isZeroAdminLogin {
		_ = uc.challengeRepo.Delete(ctx, normalized)
	}

	jwtToken, expireAt, err := token.Generate(normalized, uc.jwtSecret(), time.Now())
	if err != nil {
		return nil, "", time.Time{}, false, err
	}
	if uc.shouldBeAdmin(normalized, user) {
		_ = uc.userRepo.SetRole(ctx, user.ID, RoleAdmin)
		user.Role = RoleAdmin
	}
	return user, jwtToken, expireAt, isNewUser, nil
}

func (uc *AuthUsecase) shouldBeAdmin(address string, user *User) bool {
	if user != nil && (user.Role == RoleAdmin || user.Address == ZeroAddress) {
		return true
	}
	if address == ZeroAddress {
		return true
	}
	for _, item := range uc.auth.GetAdminAddresses() {
		if strings.EqualFold(item, address) {
			return true
		}
	}
	return false
}

func (uc *AuthUsecase) IsAdminUser(user *User) bool {
	return IsAdmin(user, uc.auth)
}

func (uc *AuthUsecase) ensureInviterActivated(ctx context.Context, inviter *User) error {
	if inviter == nil {
		return nil
	}
	if inviter.Address == ZeroAddress || uc.isBootstrapAddress(inviter.Address) {
		return nil
	}
	count, err := uc.userRepo.CountSubscribeOrders(ctx, inviter.ID)
	if err != nil {
		return err
	}
	if count <= 0 {
		return errors.BadRequest(v1.ErrorReason_INVITE_CODE_INVALID.String(), "该用户未激活")
	}
	return nil
}

func (uc *AuthUsecase) registerNewUser(ctx context.Context, address, inviteCode string) (*User, error) {
	inviteCode = strings.TrimSpace(inviteCode)
	if inviteCode == "" && !uc.isBootstrapAddress(address) && address != ZeroAddress {
		return nil, errors.BadRequest(v1.ErrorReason_INVITE_CODE_REQUIRED.String(), "首次登录需要邀请码（已登录用户的钱包地址）")
	}
	var inviter *User
	if inviteCode != "" {
		normalizedInvite, err := eth.NormalizeAddress(inviteCode)
		if err != nil {
			return nil, errors.BadRequest(v1.ErrorReason_INVITE_CODE_INVALID.String(), "邀请码格式无效")
		}
		inviter, err = uc.userRepo.FindByAddress(ctx, normalizedInvite)
		if err != nil {
			return nil, err
		}
		if inviter == nil {
			return nil, errors.BadRequest(v1.ErrorReason_INVITE_CODE_INVALID.String(), "邀请码无效，邀请人尚未登录注册")
		}
	}
	user := &User{Address: address, InviteCode: address}
	// 注册赠送只给真实用户，零地址（后台入口）与创世引导地址不参与
	if address != ZeroAddress && !uc.isBootstrapAddress(address) {
		if bonus := GetRegisterBonus(); bonus.IsPositive() {
			user.UsdtRecharge = bonus.String()
		}
	}
	if inviter != nil {
		user.InviterID = &inviter.ID
		user.InviterAddress = inviter.Address
	}
	return uc.userRepo.Create(ctx, user)
}

// resolveCaller 解析 token 所属账户并拒绝已冻结账户。
// 冻结只写数据库，签发在冻结之前的 token 仍然有效，因此每个已登录入口都要重新确认状态。
func (uc *AuthUsecase) resolveCaller(ctx context.Context, tokenString string) (*User, error) {
	address, err := token.Parse(tokenString, uc.jwtSecret())
	if err != nil {
		return nil, errors.Unauthorized(v1.ErrorReason_AUTH_UNSPECIFIED.String(), "token 无效或已过期")
	}
	user, err := uc.userRepo.FindByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "用户不存在")
	}
	if user.IsFrozen {
		return nil, errors.Forbidden(AccountFrozenReason, AccountFrozenMessage)
	}
	return user, nil
}

func (uc *AuthUsecase) GetProfile(ctx context.Context, tokenString string) (*User, int32, int32, error) {
	user, err := uc.resolveCaller(ctx, tokenString)
	if err != nil {
		return nil, 0, 0, err
	}
	count, err := uc.userRepo.CountInvitees(ctx, user.ID)
	if err != nil {
		return nil, 0, 0, err
	}
	totalDownline, err := uc.userRepo.CountUsersUnder(ctx, user.ID)
	if err != nil {
		return nil, 0, 0, err
	}
	return user, count, totalDownline, nil
}

func (uc *AuthUsecase) UpdateProfile(ctx context.Context, tokenString, username string) (*User, error) {
	user, err := uc.resolveCaller(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(username)
	if name == "" {
		return nil, errors.BadRequest(v1.ErrorReason_INVALID_ADDRESS.String(), "用户名不能为空")
	}
	if utf8.RuneCountInString(name) > 24 {
		return nil, errors.BadRequest(v1.ErrorReason_INVALID_ADDRESS.String(), "用户名不能超过24个字符")
	}
	if err := uc.userRepo.UpdateUsername(ctx, user.ID, name); err != nil {
		return nil, err
	}
	user.Username = name
	return user, nil
}

func (uc *AuthUsecase) ListInvitees(ctx context.Context, tokenString, address string) ([]*DirectInvitee, error) {
	// 冻结判定针对调用者本人；查看的目标可以是下级，不因下级被冻结而拒绝。
	caller, err := uc.resolveCaller(ctx, tokenString)
	if err != nil {
		return nil, err
	}
	targetAddress := strings.TrimSpace(address)
	if targetAddress == "" {
		targetAddress = caller.Address
	}
	normalized, err := eth.NormalizeAddress(targetAddress)
	if err != nil {
		return nil, errors.BadRequest(v1.ErrorReason_INVALID_ADDRESS.String(), "无效的钱包地址")
	}
	user := caller
	if normalized != caller.Address {
		user, err = uc.userRepo.FindByAddress(ctx, normalized)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "用户不存在")
		}
	}
	invitees, err := uc.userRepo.ListDirectInvitees(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	// 只建一次子树索引：原先对每个直推都 CountUsersUnder，会反复全表扫描用户关系，
	// 大团队请求超时后前端表现为「团队成员档案显示不全」。
	stakeMap, childrenMap, err := uc.buildPerfTree(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	inviteeIDs := make([]int64, 0, len(invitees))
	for _, item := range invitees {
		inviteeIDs = append(inviteeIDs, item.ID)
	}
	personalMap, err := uc.userRepo.SumCumulativePrincipalByUserIDs(ctx, inviteeIDs)
	if err != nil {
		return nil, err
	}
	memo := make(map[int64]decimal.Decimal)
	result := make([]*DirectInvitee, 0, len(invitees))
	for _, item := range invitees {
		directCount := int32(len(childrenMap[item.ID]))
		teamDownline := countDescendants(item.ID, childrenMap)
		lineExit := CalcSubtreeStake(item.ID, stakeMap, childrenMap, memo)
		result = append(result, &DirectInvitee{
			Address:           item.Address,
			Username:          item.Username,
			TeamStake:         item.TeamPerf,
			PersonalStake:     personalMap[item.ID],
			ExitAmount:        lineExit.String(),
			CommunityLevel:    item.CommunityLevel,
			ReleasedBalance:   item.UsdtReward,
			ShareProfitTotal:  "0",
			EcoRewardTotal:    "0",
			DirectCount:       directCount,
			TeamDownlineCount: teamDownline,
			CreatedAt:         item.CreatedTime,
		})
	}
	return result, nil
}

// countDescendants 统计 root 之下全部后代人数（不含本人）。
func countDescendants(root int64, children map[int64][]int64) int32 {
	var n int32
	queue := []int64{root}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, child := range children[cur] {
			n++
			queue = append(queue, child)
		}
	}
	return n
}

func (uc *AuthUsecase) buildPerfTree(ctx context.Context, rootID int64) (map[int64]decimal.Decimal, map[int64][]int64, error) {
	under, err := uc.userRepo.ListUsersUnder(ctx, rootID)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]int64, 0, len(under)+1)
	ids = append(ids, rootID)
	childrenMap := make(map[int64][]int64)
	for _, u := range under {
		ids = append(ids, u.ID)
		if u.InviterID != nil {
			childrenMap[*u.InviterID] = append(childrenMap[*u.InviterID], u.ID)
		}
	}
	perfMap, err := uc.userRepo.SumPrincipalByUserIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	stakeMap := make(map[int64]decimal.Decimal, len(ids))
	for _, id := range ids {
		v, _ := decimal.NewFromString(perfMap[id])
		stakeMap[id] = v
	}
	return stakeMap, childrenMap, nil
}

func randomNonce(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
