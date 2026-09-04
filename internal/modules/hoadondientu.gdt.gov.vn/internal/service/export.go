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
		file, eerr := s.exportInvoicesUpstream(
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
		if eerr != nil {
			lastErr = eerr
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
				"error", eerr,
			)

			if ctx.Err() != nil {
				return nil, s.mapToAppError(eerr)
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

type preparedMergedExport struct {
	fromDate string
	toDate   string
	ranges   []moduleutils.DateRange
	filter   model.InvoiceFilter
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

	direction, err := getDirectionFromString(req.Type)
	if err != nil {
		return nil, err
	}

	channel := model.InvoiceChannelStandard
	if req.Sco {
		channel = model.InvoiceChannelSCO
	}

	prepared, err := s.prepareMergedExport(
		req.InvoiceFilter,
		req.FromDate,
		req.ToDate,
	)
	if err != nil {
		return nil, err
	}

	token, err := s.getTokenFromSession(sessionID)
	if err != nil {
		return nil, err
	}

	return s.exportInvoiceMergedWithToken(
		ctx,
		token,
		channel,
		direction,
		prepared,
	)
}

func (s *service) ExportInvoiceSoldWrapper(
	ctx context.Context,
	sessionID string,
	req *dto.ExportInvoiceWrapperRequest,
) (
	*model.File,
	error,
) {
	return s.exportInvoiceWrapper(
		ctx,
		sessionID,
		model.InvoiceDirectionSold,
		req,
	)
}

func (s *service) ExportInvoicePurchaseWrapper(
	ctx context.Context,
	sessionID string,
	req *dto.ExportInvoiceWrapperRequest,
) (
	*model.File,
	error,
) {
	return s.exportInvoiceWrapper(
		ctx,
		sessionID,
		model.InvoiceDirectionPurchase,
		req,
	)
}

type wrapperExportResult struct {
	channel model.InvoiceChannel
	file    *model.File
	err     error
}

func (s *service) exportInvoiceWrapper(
	ctx context.Context,
	sessionID string,
	direction model.InvoiceDirection,
	req *dto.ExportInvoiceWrapperRequest,
) (
	*model.File,
	error,
) {
	if req == nil {
		return nil, apperr.New(apperr.CodeInvalidRequest, nil)
	}

	prepared, err := s.prepareMergedExport(
		req.InvoiceFilter,
		req.FromDate,
		req.ToDate,
	)
	if err != nil {
		return nil, err
	}

	// Resolve session/token đúng một lần cho toàn bộ wrapper export.
	token, err := s.getTokenFromSession(sessionID)
	if err != nil {
		return nil, err
	}

	channels := []model.InvoiceChannel{
		model.InvoiceChannelStandard,
		model.InvoiceChannelSCO,
	}
	results := make([]wrapperExportResult, 0, len(channels))

	// Export endpoints của GDT cũng chịu rate limit. Chạy tuần tự Standard/SCO;
	// từng request còn đi qua upstream limiter + retry/backoff chung.
	for _, channel := range channels {
		file, exportErr := s.exportInvoiceMergedWithToken(
			ctx,
			token,
			channel,
			direction,
			prepared,
		)
		results = append(results, wrapperExportResult{
			channel: channel,
			file:    file,
			err:     exportErr,
		})
	}

	files := make([][]byte, 0, len(results))
	mergeSources := make([]modulexlsx.MergeSource, 0, len(results))
	for _, result := range results {
		if result.err != nil {
			slog.Error(
				"HDDT GDT wrapper export source failed",
				"channel", result.channel,
				"direction", direction,
				"from", req.FromDate,
				"to", req.ToDate,
				"error", result.err,
			)
			return nil, result.err
		}
		if result.file == nil || len(result.file.Body) == 0 {
			return nil, apperr.New(
				apperr.CodeHDDTGDTInvalidResponse,
				fmt.Errorf("HDDT GDT wrapper export returned empty %s file", result.channel),
			)
		}

		files = append(files, result.file.Body)
		mergeSources = append(mergeSources, modulexlsx.MergeSource{
			Name: wrapperExportSourceName(result.channel),
			Body: result.file.Body,
		})
	}

	// Cùng channel (các tháng) dùng chung schema nên strict Merge giữ nguyên
	// template GDT. Nhưng Standard và SCO có thể khác số lượng/tên cột. Không
	// được bỏ validate rồi ghép theo vị trí vì sẽ làm lệch dữ liệu.
	body, strictMergeErr := modulexlsx.Merge(files)
	if strictMergeErr != nil {
		slog.Warn(
			"HDDT GDT wrapper export schemas differ; using header merge",
			"direction", direction,
			"from", req.FromDate,
			"to", req.ToDate,
			"error", strictMergeErr,
		)

		body, err = modulexlsx.MergeByHeader(
			mergeSources,
			wrapperExportTitle(direction),
		)
		if err != nil {
			slog.Error(
				"HDDT GDT wrapper export header merge failed",
				"direction", direction,
				"from", req.FromDate,
				"to", req.ToDate,
				"strict_error", strictMergeErr,
				"error", err,
			)
			return nil, apperr.New(
				apperr.CodeHDDTGDTInvalidResponse,
				fmt.Errorf(
					"merge standard and SCO export files: strict merge: %v; header merge: %w",
					strictMergeErr,
					err,
				),
			)
		}
	}

	return &model.File{
		Body:        body,
		ContentType: modulexlsx.ContentType(),
		Filename: fmt.Sprintf(
			"hddtgdt-%s-%s_%s.xlsx",
			direction,
			req.FromDate,
			req.ToDate,
		),
	}, nil
}

func wrapperExportSourceName(channel model.InvoiceChannel) string {
	switch channel {
	case model.InvoiceChannelStandard:
		return "Hóa đơn thường"
	case model.InvoiceChannelSCO:
		return "Máy tính tiền"
	default:
		return string(channel)
	}
}

func wrapperExportTitle(direction model.InvoiceDirection) string {
	if direction == model.InvoiceDirectionPurchase {
		return "Hóa đơn mua vào"
	}
	return "Hóa đơn bán ra"
}

func (s *service) prepareMergedExport(
	invoiceFilter dto.InvoiceFilter,
	fromDate string,
	toDate string,
) (
	*preparedMergedExport,
	error,
) {
	from, to, err := moduleutils.ParseDateRange(fromDate, toDate)
	if err != nil {
		return nil, apperr.New(apperr.CodeInvalidRequest, err)
	}

	filter := mapToFilter(invoiceFilter)
	if _, err := moduleutils.BuildSearch(from, to, filter); err != nil {
		return nil, apperr.New(apperr.CodeInvalidRequest, err)
	}

	return &preparedMergedExport{
		fromDate: fromDate,
		toDate:   toDate,
		ranges: moduleutils.SplitDateRangeDescending(
			from,
			to,
			s.maxExportDays,
		),
		filter: filter,
	}, nil
}

func (s *service) exportInvoiceMergedWithToken(
	ctx context.Context,
	token string,
	channel model.InvoiceChannel,
	direction model.InvoiceDirection,
	prepared *preparedMergedExport,
) (
	*model.File,
	error,
) {
	if prepared == nil {
		return nil, apperr.New(apperr.CodeInvalidRequest, nil)
	}

	files := make([][]byte, 0, len(prepared.ranges))
	for _, dateRange := range prepared.ranges {
		file, err := s.exportInvoicesUpstream(
			ctx,
			token,
			channel,
			direction,
			model.ExportOptions{
				From:   dateRange.From,
				To:     dateRange.To,
				Filter: prepared.filter,
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

		if validateErr := modulexlsx.Validate(file.Body); validateErr != nil {
			slog.Error(
				"HDDT GDT export returned unreadable workbook",
				"channel", channel,
				"direction", direction,
				"from", dateRange.From,
				"to", dateRange.To,
				"content_type", file.ContentType,
				"body_size", len(file.Body),
				"error", validateErr,
			)
			return nil, apperr.New(
				apperr.CodeHDDTGDTInvalidResponse,
				fmt.Errorf(
					"invalid export workbook for %s %s to %s: %w",
					channel,
					moduleutils.FormatInputDate(dateRange.From),
					moduleutils.FormatInputDate(dateRange.To),
					validateErr,
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
			"hddtgdt-%s-%s-merged-%s_%s.xlsx",
			channel,
			direction,
			prepared.fromDate,
			prepared.toDate,
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
			"hddtgdt-%s-%s-chunks-%s_%s.zip",
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
		"hddtgdt-%s-%s-%s_%s.xlsx",
		channel,
		direction,
		moduleutils.FormatInputDate(from),
		moduleutils.FormatInputDate(to),
	)
}
