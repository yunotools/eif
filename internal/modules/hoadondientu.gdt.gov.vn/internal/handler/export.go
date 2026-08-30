package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yunotools/eif/internal/core/apperr"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
)

func (h *Handler) ExportInvoice(c *gin.Context) {
	var req dto.ExportInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperr.New(apperr.CodeInvalidRequest, err))
		return
	}

	file, err := h.service.ExportInvoice(
		c.Request.Context(),
		sessionID(c),
		&req,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(
		"attachment; filename=%q",
		file.Filename,
	))
	c.Data(http.StatusOK, file.ContentType, file.Body)
}

func (h *Handler) ExportInvoiceMerged(c *gin.Context) {
	var req dto.ExportInvoiceMergedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		))
		return
	}

	file, err := h.service.ExportInvoiceMerged(
		c.Request.Context(),
		sessionID(c),
		&req,
	)
	if err != nil {
		writeError(c, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(
		"attachment; filename=%q",
		file.Filename,
	))
	c.Data(http.StatusOK, file.ContentType, file.Body)
}
