package service

import (
	"errors"
	"net"
	"net/http"

	"github.com/yunotools/eif/internal/core/apperr"
	corehttp "github.com/yunotools/eif/internal/core/protocol/httpclient"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/client"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/session"
)

type service struct {
	client        client.Client
	sessions      *session.Manager
	maxQueryDays  int
	maxExportDays int
}

func New(
	client client.Client,
	sessions *session.Manager,
	maxQueryDays,
	maxExportDays int,
) Service {
	return &service{
		client:        client,
		sessions:      sessions,
		maxQueryDays:  maxQueryDays,
		maxExportDays: maxExportDays,
	}
}

func (s *service) getTokenFromSession(
	sessionID string,
) (
	string,
	error,
) {
	sess, err := s.sessions.Get(sessionID)
	if err != nil {
		return "", apperr.New(
			apperr.CodeSessionExpired,
			err,
		)
	}

	return sess.Token, nil
}

func (s *service) mapToAppError(err error) error {
	if err == nil {
		return nil
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return apperr.New(
			apperr.CodeHDDTGDTTimeout,
			err,
		)
	}

	var httpErr *corehttp.HTTPError
	if errors.As(err, &httpErr) {
		if httpErr.StatusCode == http.StatusUnauthorized ||
			httpErr.StatusCode == http.StatusForbidden {
			return apperr.New(apperr.CodeSessionExpired, err)
		}
		return apperr.New(apperr.CodeHDDTGDTBadGateway, err)
	}

	return apperr.New(apperr.CodeHDDTGDTInvalidResponse, err)
}
