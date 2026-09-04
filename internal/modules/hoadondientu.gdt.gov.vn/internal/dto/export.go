package dto

type ExportInvoiceRequest struct {
	InvoiceFilter
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
	Type     string `json:"type"`
	Sco      bool   `json:"sco"`
}

type ExportInvoiceMergedRequest struct {
	InvoiceFilter
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
	Type     string `json:"type"`
	Sco      bool   `json:"sco"`
}

type ExportInvoiceWrapperRequest struct {
	InvoiceFilter
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
}
