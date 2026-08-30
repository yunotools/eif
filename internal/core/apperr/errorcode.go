package apperr

import "net/http"

type ErrorCode struct {
	Code       string
	StatusCode int
	Message    string
}

var (
	CodeInternalError = ErrorCode{
		Code:       "EIF-SYS-500",
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal error",
	}

	CodeInvalidRequest = ErrorCode{
		Code:       "EIF-REQ-400",
		StatusCode: http.StatusBadRequest,
		Message:    "Invalid request",
	}

	CodeUnauthorized = ErrorCode{
		Code:       "EIF-AUTH-401",
		StatusCode: http.StatusUnauthorized,
		Message:    "Unauthorized",
	}

	CodeSessionExpired = ErrorCode{
		Code:       "EIF-AUTH-SESSION-401",
		StatusCode: http.StatusUnauthorized,
		Message:    "Session is missing or expired",
	}

	CodeHDDTGDTAuthenticationFailed = ErrorCode{
		Code:       "EIF-HDDT-GDT-AUTH-401",
		StatusCode: http.StatusUnauthorized,
		Message:    "HDDT GDT authentication failed",
	}

	CodeHDDTGDTTimeout = ErrorCode{
		Code:       "EIF-HDDT-GDT-504",
		StatusCode: http.StatusGatewayTimeout,
		Message:    "HDDT GDT timeout",
	}

	CodeHDDTGDTBadGateway = ErrorCode{
		Code:       "EIF-HDDT-GDT-502",
		StatusCode: http.StatusBadGateway,
		Message:    "HDDT GDT request failed",
	}

	CodeHDDTGDTInvalidResponse = ErrorCode{
		Code:       "EIF-HDDT-GDT-INVALID-502",
		StatusCode: http.StatusBadGateway,
		Message:    "HDDT GDT returned an invalid response",
	}

	CodeExportRangeTooLarge = ErrorCode{
		Code:       "EIF-HDDT-GDT-EXPORT-400",
		StatusCode: http.StatusBadRequest,
		Message:    "Export date range is too large",
	}
)
