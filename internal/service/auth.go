package service

import (
	"context"
	"net/http"
	"strings"

	v1 "backend/api/auth/v1"
	"backend/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// AuthService implements auth.v1.AuthServer.
type AuthService struct {
	v1.UnimplementedAuthServer

	uc           *biz.AuthUsecase
	loginDevices biz.LoginDeviceRepo
	log          *log.Helper
}

// NewAuthService creates AuthService.
func NewAuthService(uc *biz.AuthUsecase, loginDevices biz.LoginDeviceRepo, logger log.Logger) *AuthService {
	return &AuthService{
		uc:           uc,
		loginDevices: loginDevices,
		log:          log.NewHelper(logger),
	}
}

func (s *AuthService) GetChallenge(ctx context.Context, req *v1.GetChallengeRequest) (*v1.GetChallengeReply, error) {
	challenge, err := s.uc.GetChallenge(ctx, req.Address)
	if err != nil {
		return nil, err
	}
	return &v1.GetChallengeReply{
		Message:  challenge.Message,
		ExpireAt: challenge.ExpireAt.Unix(),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginReply, error) {
	user, token, expireAt, isNewUser, err := s.uc.Login(ctx, req.Address, req.Signature, req.InviteCode)
	if err != nil {
		return nil, err
	}
	s.recordLoginDevice(ctx, user.ID, req)
	return &v1.LoginReply{
		Token:          token,
		ExpireAt:       expireAt.Unix(),
		IsNewUser:      isNewUser,
		Address:        user.Address,
		InviterAddress: user.InviterAddress,
		IsAdmin:        s.uc.IsAdminUser(user),
	}, nil
}

func (s *AuthService) recordLoginDevice(ctx context.Context, userID int64, req *v1.LoginRequest) {
	if s.loginDevices == nil || userID <= 0 {
		return
	}
	info := biz.LoginDeviceInfo{
		DeviceID: strings.TrimSpace(req.DeviceID),
		Client:   strings.TrimSpace(req.Client),
	}
	if r := httpRequestFromContext(ctx); r != nil {
		info.ClientIP = clientIP(r)
		info.UserAgent = r.UserAgent()
	}
	if err := s.loginDevices.RecordLogin(ctx, userID, info); err != nil {
		s.log.Warnf("record login device user=%d: %v", userID, err)
	}
}

func httpRequestFromContext(ctx context.Context) *http.Request {
	if req, ok := khttp.RequestFromServerContext(ctx); ok && req != nil {
		return req
	}
	tr, ok := transport.FromServerContext(ctx)
	if !ok || tr == nil {
		return nil
	}
	ht, ok := tr.(*khttp.Transport)
	if !ok || ht == nil {
		return nil
	}
	return ht.Request()
}

func (s *AuthService) GetProfile(ctx context.Context, req *v1.GetProfileRequest) (*v1.GetProfileReply, error) {
	user, inviteeCount, totalDownline, err := s.uc.GetProfile(ctx, resolveToken(ctx, req.Token))
	if err != nil {
		return nil, err
	}
	return &v1.GetProfileReply{
		Address:            user.Address,
		InviterAddress:     user.InviterAddress,
		InviteeCount:       inviteeCount,
		CreatedAt:          user.CreatedAt.Unix(),
		DownlineInvitees:   nil,
		TotalDownlineCount: totalDownline,
		CommunityLevel:     user.CommunityLevel,
		CommunityStake:     user.CommunityStake,
		TeamStake:          user.TeamStake,
		ShareProfitTotal:   user.ShareProfitTotal,
		EcoRewardTotal:     user.EcoRewardTotal,
		IsAdmin:            s.uc.IsAdminUser(user),
		Username:           user.Username,
	}, nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, req *v1.UpdateProfileRequest) (*v1.UpdateProfileReply, error) {
	user, err := s.uc.UpdateProfile(ctx, resolveToken(ctx, req.Token), req.Username)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateProfileReply{Username: user.Username}, nil
}

func (s *AuthService) ListInvitees(ctx context.Context, req *v1.ListInviteesRequest) (*v1.ListInviteesReply, error) {
	invitees, err := s.uc.ListInvitees(ctx, resolveToken(ctx, req.Token), req.Address)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.InviteeNode, 0, len(invitees))
	for _, item := range invitees {
		items = append(items, &v1.InviteeNode{
			Address:           item.Address,
			Username:          item.Username,
			TeamStake:         item.TeamStake,
			PersonalStake:     item.PersonalStake,
			ReleasedBalance:   item.ReleasedBalance,
			ShareProfitTotal:  item.ShareProfitTotal,
			EcoRewardTotal:    item.EcoRewardTotal,
			ExitAmount:        item.ExitAmount,
			CommunityLevel:    item.CommunityLevel,
			DirectCount:       item.DirectCount,
			TeamDownlineCount: item.TeamDownlineCount,
			CreatedAt:         item.CreatedAt.Unix(),
		})
	}
	return &v1.ListInviteesReply{Invitees: items}, nil
}
