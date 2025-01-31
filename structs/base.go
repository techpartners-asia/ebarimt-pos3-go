package structs

import (
	"time"

	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	"gorm.io/gorm"
)

type (
	Order struct {
		ID            uint    `json:"id" gorm:"primary_key;auto_increment"`
		BranchID      uint    `gorm:"column:branch_id;index" json:"branch_id"`        // Салбарын мэдээлэл
		TotalPrice    float64 
		IsOrg bool 
		OrgRegNo string
		MerchantTin string 
		BankAccountNo string
		Type constants.ReceiptType
		Items         []OrderItem `gorm:"foreignKey:OrderID" json:"items"` // Хүргэлтийн
	}
	OrderItem struct {
		ID uint
		ProductName string
		OrderID            uint    `gorm:"column:order_id;index" json:"order_id"`
		PriceTotal         float64 `gorm:"column:price_total" json:"price_total"`
		ClassificationCode string
		TaxProductCode     string
		TaxType            constants.TaxType 
		IsCityTax bool 
		Qty                float64     `gorm:"column:qty" json:"qty"`
		PriceUnit          float64 `gorm:"column:price_unit" json:"price_unit"`
		Note               string  `gorm:"column:note" json:"note"`
	}
	EbarimtStock struct {
		ID        uint           `gorm:"primary_key;autoIncrement:true" json:"id"`
		CreatedAt time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
		UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
		DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
		OrderItemID        uint   `gorm:"column:order_item_id;index" json:"order_item_id"`
		EbarimtID          uint   `gorm:"column:ebarimt_id;index" json:"ebarimt_id"`
		Code               string `gorm:"column:code" json:"code"`
		Name               string `gorm:"column:name" json:"name"`
		MeasureUnit        string `gorm:"column:measureUnit" json:"measureUnit"`
		Qty                string `gorm:"column:qty" json:"qty"`
		UnitPrice          string `gorm:"column:unitPrice" json:"unitPrice"`
		TotalAmount        string `gorm:"column:totalAmount" json:"totalAmount"`
		CityTax            string `gorm:"column:cityTax" json:"cityTax"`
		Vat                string `gorm:"column:vat" json:"vat"`
		BarCode            string `gorm:"column:barCode" json:"barCode"`
		ClassificationCode string `gorm:"column:classification_code" json:"classification_code"`
		TaxProductCode     string `gorm:"column:tax_product_code" json:"tax_product_code"`
		TaxType            string `gorm:"column:tax_type" json:"tax_type"`
	}
	Ebarimt struct {
		ID        uint           `gorm:"primary_key;autoIncrement:true" json:"id"`
		CreatedAt time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
		UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
		DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
		Stocks        []EbarimtStock `gorm:"foreignKey:EbarimtID" json:"stocks"`
		OrderID       uint           `gorm:"column:order_id;index" json:"order_id"`
		Order         *Order         `gorm:"foreignKey:OrderID" json:"order,omitempty"`
		Amount        string         `gorm:"column:amount" json:"amount"`
		Vat           string         `gorm:"column:vat" json:"vat"`
		CashAmount    string         `gorm:"column:cash_amount" json:"cash_amount"`
		NonCashAmount string         `gorm:"column:non_cash_amount" json:"non_cash_amount"`
		CityTax       string         `gorm:"column:city_tax" json:"city_tax"`
		CustomerNo    string         `gorm:"column:customer_no" json:"customer_no"`
		BillType      string         `gorm:"column:bill_type" json:"bill_type"`
		BranchNo      string         `gorm:"column:branch_no" json:"branch_no"`
		DistrictCode  string         `gorm:"column:district_code" json:"district_code"`
		TaxType       string         `gorm:"column:tax_type" json:"tax_type"`
		RegisterNo    string         `gorm:"column:register_no" json:"register_no"`
		BillId        string         `gorm:"column:bill_id" json:"bill_id"`
		MacAddress    string         `gorm:"column:mac_address" json:"mac_address"`
		Date          string         `gorm:"column:date" json:"date"`
		Lottery       string         `gorm:"column:lottery" json:"lottery"`
		InternalCode  string         `gorm:"column:internal_code" json:"internal_code"`
		QrData        string         `gorm:"column:qr_data" json:"qr_data"`
		MerchantId    string         `gorm:"column:merchant_id" json:"merchant_id"`
		Success       bool           `gorm:"column:success" json:"success"`
		IsCancelled   bool           `gorm:"column:is_cancelled" json:"is_cancelled"`
		Message       string         `gorm:"column:message" json:"message"`
	}
)