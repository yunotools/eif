package service

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/yunotools/eif/internal/core/apperr"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/session"
)

func (s *service) GetCaptcha(
	ctx context.Context,
) (
	*dto.CaptchaResponse,
	error,
) {
	response, err := s.client.GetCaptcha(ctx)
	if err != nil {
		return nil, s.mapToAppError(err)
	}

	return response, nil
}

func (s *service) Authenticate(
	ctx context.Context,
	req *dto.AuthenticationRequest,
) (
	*dto.AuthenticationResponse,
	error,
) {
	if req == nil ||
		req.Username == "" ||
		req.Password == "" ||
		req.CValue == "" ||
		req.CKey == "" {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			errors.New("username, password, cvalue and ckey are required"),
		)
	}

	response, err := s.client.Authenticate(ctx, req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, apperr.New(
				apperr.CodeHDDTGDTTimeout,
				err,
			)
		}
		return nil, apperr.New(
			apperr.CodeHDDTGDTAuthenticationFailed,
			err,
		)
	}
	if response == nil || response.Token == "" {
		return nil, apperr.New(
			apperr.CodeHDDTGDTInvalidResponse,
			errors.New("authenticate response does not contain token"),
		)
	}

	expiredAt, err := session.ParseJWTExpiry(response.Token)
	if err != nil {
		return nil, apperr.New(
			apperr.CodeHDDTGDTInvalidResponse,
			err,
		)
	}

	created, err := s.sessions.Create(
		req.Username,
		response.Token,
		expiredAt,
	)
	if err != nil {
		return nil, apperr.New(
			apperr.CodeInternalError,
			err,
		)
	}

	return &dto.AuthenticationResponse{
		SessionID: created.ID,
		ExpiredAt: created.ExpiredAt,
	}, nil
}

func (s *service) GetSession(sessionID string) (
	*dto.SessionResponse,
	error,
) {
	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return nil, apperr.New(
			apperr.CodeSessionExpired,
			err,
		)
	}

	remaining := time.Until(sess.ExpiredAt)
	remainingSeconds := int64(0)
	if remaining > 0 {
		// Làm tròn lên để vừa nhận response không bị hụt 1 giây do phần nano.
		remainingSeconds = int64(
			(remaining + time.Second - 1) / time.Second,
		)
	}

	return &dto.SessionResponse{
		SessionID:        sess.ID,
		Username:         sess.Username,
		ExpiredAt:        sess.ExpiredAt,
		RemainingSeconds: remainingSeconds,
	}, nil
}

func (s *service) DeleteSession(sessionID string) error {
	if sessionID == "" {
		return nil
	}

	if err := s.sessions.Delete(sessionID); err != nil {
		return apperr.New(
			apperr.CodeInternalError,
			err,
		)
	}

	return nil
}
