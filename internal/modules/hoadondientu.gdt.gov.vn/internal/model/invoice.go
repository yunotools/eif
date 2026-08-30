package model

import (
	"encoding/json"
	"time"
)

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

type InvoiceQueryResult struct {
	Datas []json.RawMessage `json:"datas"`
	Total int               `json:"total"`
	State any               `json:"state"`
	Time  int               `json:"time"`
}

type File struct {
	Body        []byte
	ContentType string
	Filename    string
}
