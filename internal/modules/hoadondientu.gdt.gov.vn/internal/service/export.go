package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/yunotools/eif/internal/core/apperr"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/dto"
	"github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/model"
	moduleutils "github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/utils"
	modulexlsx "github.com/yunotools/eif/internal/modules/hoadondientu.gdt.gov.vn/internal/xlsx"
)

const zipContentType = "application/zip"

type exportedInvoiceChunk struct {
	DateRange moduleutils.DateRange
	File      *model.File
}

type exportFailedRangesManifest struct {
	FromDate     string                          `json:"from_date"`
	ToDate       string                          `json:"to_date"`
	FailedRanges []model.InvoiceQueryFailedRange `json:"failed_ranges"`
}

func (s *service) ExportInvoice(
	ctx context.Context,
	sessionID string,
	req *dto.ExportInvoiceRequest,
) (
	*model.File,
	error,
) {
	if req == nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			nil,
		)
	}

	token, err := s.getTokenFromSession(sessionID)
	if err != nil {
		return nil, err
	}

	direction, err := getDirectionFromString(req.Type)
	if err != nil {
		return nil, err
	}

	channel := model.InvoiceChannelStandard
	if req.Sco {
		channel = model.InvoiceChannelSCO
	}

	from, to, err := moduleutils.ParseDateRange(
		req.FromDate,
		req.ToDate,
	)
	if err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	filter := mapToFilter(req.InvoiceFilter)
	if _, err := moduleutils.BuildSearch(from, to, filter); err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	ranges := moduleutils.SplitDateRangeDescending(
		from,
		to,
		s.maxExportDays,
	)

	exported := make([]exportedInvoiceChunk, 0, len(ranges))
	failedRanges := make([]model.InvoiceQueryFailedRange, 0)
	var lastErr error

	for _, dateRange := range ranges {
		file, err := s.client.ExportInvoices(
			ctx,
			token,
			channel,
			direction,
			model.ExportOptions{
				From:   dateRange.From,
				To:     dateRange.To,
				Filter: filter,
			},
		)
		if err != nil {
			lastErr = err
			failedRanges = append(
				failedRanges,
				model.InvoiceQueryFailedRange{
					FromDate: moduleutils.FormatInputDate(dateRange.From),
					ToDate:   moduleutils.FormatInputDate(dateRange.To),
				},
			)

			slog.Error(
				"HDDT GDT export request failed",
				"channel", channel,
				"direction", direction,
				"from", dateRange.From,
				"to", dateRange.To,
				"error", err,
			)

			if ctx.Err() != nil {
				return nil, s.mapToAppError(err)
			}

			continue
		}

		if file == nil || len(file.Body) == 0 {
			lastErr = fmt.Errorf("HDDT GDT returned empty export file")
			failedRanges = append(
				failedRanges,
				model.InvoiceQueryFailedRange{
					FromDate: moduleutils.FormatInputDate(dateRange.From),
					ToDate:   moduleutils.FormatInputDate(dateRange.To),
				},
			)
			continue
		}

		if file.ContentType == "" {
			file.ContentType = modulexlsx.ContentType()
		}
		file.Filename = exportChunkFilename(
			channel,
			direction,
			dateRange.From,
			dateRange.To,
		)

		exported = append(exported, exportedInvoiceChunk{
			DateRange: dateRange,
			File:      file,
		})
	}

	if len(exported) == 0 {
		if lastErr == nil {
			lastErr = fmt.Errorf("HDDT GDT export did not return any file")
		}
		return nil, s.mapToAppError(lastErr)
	}

	// Giữ behavior cũ cho request chỉ có đúng một chunk thành công:
	// trả file XLSX trực tiếp, không bọc ZIP.
	if len(ranges) == 1 && len(failedRanges) == 0 {
		return exported[0].File, nil
	}

	return buildExportChunksZip(
		channel,
		direction,
		req.FromDate,
		req.ToDate,
		exported,
		failedRanges,
	)
}

func (s *service) ExportInvoiceMerged(
	ctx context.Context,
	sessionID string,
	req *dto.ExportInvoiceMergedRequest,
) (
	*model.File,
	error,
) {
	if req == nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			nil,
		)
	}

	token, err := s.getTokenFromSession(sessionID)
	if err != nil {
		return nil, err
	}

	direction, err := getDirectionFromString(req.Type)
	if err != nil {
		return nil, err
	}

	channel := model.InvoiceChannelStandard
	if req.Sco {
		channel = model.InvoiceChannelSCO
	}

	from, to, err := moduleutils.ParseDateRange(
		req.FromDate,
		req.ToDate,
	)
	if err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	filter := mapToFilter(req.InvoiceFilter)
	if _, err := moduleutils.BuildSearch(from, to, filter); err != nil {
		return nil, apperr.New(
			apperr.CodeInvalidRequest,
			err,
		)
	}

	ranges := moduleutils.SplitDateRangeDescending(
		from,
		to,
		s.maxExportDays,
	)

	files := make([][]byte, 0, len(ranges))
	for _, dateRange := range ranges {
		file, err := s.client.ExportInvoices(
			ctx,
			token,
			channel,
			direction,
			model.ExportOptions{
				From:   dateRange.From,
				To:     dateRange.To,
				Filter: filter,
			},
		)
		if err != nil {
			slog.Error(
				"HDDT GDT merged export request failed",
				"channel", channel,
				"direction", direction,
				"from", dateRange.From,
				"to", dateRange.To,
				"error", err,
			)
			return nil, s.mapToAppError(err)
		}

		if file == nil || len(file.Body) == 0 {
			return nil, apperr.New(
				apperr.CodeHDDTGDTInvalidResponse,
				fmt.Errorf(
					"HDDT GDT returned empty export file for range %s to %s",
					moduleutils.FormatInputDate(dateRange.From),
					moduleutils.FormatInputDate(dateRange.To),
				),
			)
		}

		files = append(files, file.Body)
	}

	body, err := modulexlsx.Merge(files)
	if err != nil {
		return nil, apperr.New(
			apperr.CodeHDDTGDTInvalidResponse,
			fmt.Errorf("merge HDDT GDT export files: %w", err),
		)
	}

	return &model.File{
		Body:        body,
		ContentType: modulexlsx.ContentType(),
		Filename: fmt.Sprintf(
			"gdt-%s-%s-merged-%s_%s.xlsx",
			channel,
			direction,
			req.FromDate,
			req.ToDate,
		),
	}, nil
}

func buildExportChunksZip(
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	fromDate string,
	toDate string,
	exported []exportedInvoiceChunk,
	failedRanges []model.InvoiceQueryFailedRange,
) (
	*model.File,
	error,
) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for _, chunk := range exported {
		// XLSX bản thân đã là ZIP nên Store sẽ tránh nén lại lần hai.
		entry, err := writer.CreateHeader(&zip.FileHeader{
			Name:   chunk.File.Filename,
			Method: zip.Store,
		})
		if err != nil {
			return nil, apperr.New(
				apperr.CodeInternalError,
				err,
			)
		}
		if _, err := entry.Write(chunk.File.Body); err != nil {
			return nil, apperr.New(
				apperr.CodeInternalError,
				err,
			)
		}
	}

	if len(failedRanges) > 0 {
		manifest, err := json.MarshalIndent(exportFailedRangesManifest{
			FromDate:     fromDate,
			ToDate:       toDate,
			FailedRanges: failedRanges,
		}, "", "  ")
		if err != nil {
			return nil, apperr.New(
				apperr.CodeInternalError,
				err,
			)
		}

		entry, err := writer.Create("failed_ranges.json")
		if err != nil {
			return nil, apperr.New(
				apperr.CodeInternalError,
				err,
			)
		}

		if _, err := entry.Write(manifest); err != nil {
			return nil, apperr.New(
				apperr.CodeInternalError,
				err,
			)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, apperr.New(
			apperr.CodeInternalError,
			err,
		)
	}

	return &model.File{
		Body:        buffer.Bytes(),
		ContentType: zipContentType,
		Filename: fmt.Sprintf(
			"gdt-%s-%s-chunks-%s_%s.zip",
			channel,
			direction,
			fromDate,
			toDate,
		),
	}, nil
}

func exportChunkFilename(
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	from time.Time,
	to time.Time,
) string {
	return fmt.Sprintf(
		"gdt-%s-%s-%s_%s.xlsx",
		channel,
		direction,
		moduleutils.FormatInputDate(from),
		moduleutils.FormatInputDate(to),
	)
}
