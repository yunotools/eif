package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yunotools/eif/internal/core/apperr"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/model"
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

	writeFile(c, file)
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

	writeFile(c, file)
}

func (h *Handler) ExportInvoiceSoldWrapper(c *gin.Context) {
	h.exportWrapper(c, h.service.ExportInvoiceSoldWrapper)
}

func (h *Handler) ExportInvoicePurchaseWrapper(c *gin.Context) {
	h.exportWrapper(c, h.service.ExportInvoicePurchaseWrapper)
}

func (h *Handler) exportWrapper(
	c *gin.Context,
	fn func(context.Context, string, *dto.ExportInvoiceWrapperRequest) (*model.File, error),
) {
	var req dto.ExportInvoiceWrapperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperr.New(apperr.CodeInvalidRequest, err))
		return
	}

	file, err := fn(c.Request.Context(), sessionID(c), &req)
	if err != nil {
		writeError(c, err)
		return
	}

	writeFile(c, file)
}

func writeFile(c *gin.Context, file *model.File) {
	c.Header("Content-Disposition", fmt.Sprintf(
		"attachment; filename=%q",
		file.Filename,
	))
	c.Data(http.StatusOK, file.ContentType, file.Body)
}
