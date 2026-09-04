package model

import (
	"encoding/json"
	"time"
)

const MaxInvoiceQuerySize = 50

type InvoiceChannel string

const (
	InvoiceChannelStandard InvoiceChannel = "standard"
	InvoiceChannelSCO      InvoiceChannel = "sco"
)

type InvoiceDirection string

const (
	InvoiceDirectionSold     InvoiceDirection = "sold"
	InvoiceDirectionPurchase InvoiceDirection = "purchase"
)

type InvoiceFilter struct {
	SoHoaDon         *int64
	KyHieuHoaDon     string
	KyHieuMauSo      *int
	MaSoThueNguoiBan string
	MaSoThueNguoiMua string
	TrangThaiHoaDon  *int
	KetQuaXuLy       *int
	HoaDonUyNhiem    *int
	CanCuocCongDan   string
}

type QueryOptions struct {
	From   time.Time
	To     time.Time
	Size   int
	Filter InvoiceFilter
}

type ExportOptions struct {
	From   time.Time
	To     time.Time
	Filter InvoiceFilter
}

type InvoiceQueryFailedRange struct {
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
}

type InvoicePagination struct {
	Page        int  `json:"page"`
	PageSize    int  `json:"page_size"`
	TotalPages  int  `json:"total_pages"`
	HasPrevious bool `json:"has_previous"`
	HasNext     bool `json:"has_next"`
	Truncated   bool `json:"truncated"`
}

type InvoiceQueryResult struct {
	FromDate     string                    `json:"from_date"`
	ToDate       string                    `json:"to_date"`
	FailedRanges []InvoiceQueryFailedRange `json:"failed_ranges"`
	Datas        []json.RawMessage         `json:"datas"`
	Total        int                       `json:"total"`
	State        any                       `json:"state"`
	Time         int                       `json:"time"`
	Pagination   *InvoicePagination        `json:"pagination,omitempty"`
}

type File struct {
	Body        []byte
	ContentType string
	Filename    string
}
