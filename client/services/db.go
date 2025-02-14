package ebarimt3SdkServices

import (
	"fmt"
	"time"

	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"gorm.io/gorm"
)

// * NOTE * : Database Models -
type (
	Base struct {
		ID        int64          `gorm:"primary_key;autoIncrement:true" json:"id"`
		CreatedAt time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
		UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
		DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	}

	Ebarimt struct {
		Base
		TotalAmount  float64               `gorm:"total_amount" json:"total_amount"`
		TotalVat     float64               `gorm:"total_vat" json:"total_vat"`
		TotalCityTax float64               `gorm:"total_city_tax" json:"total_city_tax"`
		BranchNo     string                `gorm:"branch_no" json:"branch_no"`
		DistrictCode string                `gorm:"district_code" json:"district_code"`
		MerchantTin  string                `gorm:"merchant_tin" json:"merchant_tin"`
		PosNo        string                `gorm:"pos_no" json:"pos_no"`
		CustomerTin  string                `gorm:"customer_tin" json:"customer_tin"`
		ConsumerNo   string                `gorm:"consumer_no" json:"consumer_no"`
		Type         string `gorm:"type:varchar(20);column:receipt_type" json:"type"`
		BillID       string                `gorm:"bill_id" json:"bill_id"`
		InvoiceID    string                `gorm:"invoice_id" json:"invoice_id"`
		PosID        float64               `gorm:"pos_id" json:"pos_id"`
		Message      string                `gorm:"message" json:"message"`
		QrData       string                `gorm:"qr_data" json:"qr_data"`
		Lottery      string                `gorm:"lottery" json:"lottery"`
		Date         string                `gorm:"date" json:"date"`
		IsRefund     bool                  `gorm:"is_refund" json:"is_refund"`
		Receipts     []EbarimtReceipt      `gorm:"foreignKey:EbarimtID" json:"receipts"`
	}

	EbarimtReceipt struct {
		Base
		EbarimtID     int64                `gorm:"ebarimt_id" json:"ebarimt_id"`
		BillID        string               `gorm:"bill_id" json:"bill_id"`
		TotalAmount   float64              `gorm:"total_amount" json:"total_amount"`
		TotalVat      float64              `gorm:"total_vat" json:"total_vat"`
		TotalCityTax  float64              `gorm:"total_city_tax" json:"total_city_tax"`
		TaxType       string    `gorm:"tax_type" json:"tax_type"`
		MerchantTin   string               `gorm:"merchant_tin" json:"merchant_tin"`
		BankAccountNo string               `gorm:"bank_account_no" json:"bank_account_no"`
		Items         []EbarimtReceiptItem `gorm:"foreignKey:ReceiptID" json:"items"`
		IsRefund      bool                 `gorm:"is_refund" json:"is_refund"`
	}

	EbarimtReceiptItem struct {
		Base
		Name               string                `gorm:"name" json:"name"`
		BarCode            string                `gorm:"bar_code" json:"bar_code"`
		BarCodeType       string `gorm:"bar_code_type" json:"bar_code_type"`
		ClassificationCode string                `gorm:"classification_code" json:"classification_code"`
		MeasureUnit        string                `gorm:"measure_unit" json:"measure_unit"`
		TaxProductCode     string                `gorm:"tax_product_code" json:"tax_product_code"`
		Qty                float64               `gorm:"qty" json:"qty"`
		UnitPrice          float64               `gorm:"unit_price" json:"unit_price"`
		TotalAmount        float64               `gorm:"total_amount" json:"total_amount"`
		TotalVat           float64               `gorm:"total_vat" json:"total_vat"`
		TotalCityTax       float64               `gorm:"total_city_tax" json:"total_city_tax"`
		TotalBonus         float64               `gorm:"total_bonus" json:"total_bonus"`
		ReceiptID          int64                 `gorm:"receipt_id" json:"receipt_id"`
		Receipt            *EbarimtReceipt       `gorm:"foreignKey:ReceiptID" json:"receipt"`
	}
)

func Register(db *gorm.DB) {
	if err := db.AutoMigrate(&Ebarimt{}, &EbarimtReceipt{}, &EbarimtReceiptItem{}); err != nil {
		fmt.Println("Ebarimt SDK DB Migration Error :%v", err.Error())
	}
}

func SaveEbarimt(db *gorm.DB, res *structs.ReceiptResponse) {
	// Ebarimt хадгалах
	instance := Ebarimt{
		TotalAmount:  res.TotalAmount,
		TotalVat:     res.TotalAmount,
		TotalCityTax: res.TotalCityTax,
		CustomerTin:  res.CustomerTIN,
		Type:         string(res.Type),
		BranchNo:     res.BranchNo,
		DistrictCode: res.DistrictCode,
		BillID:       res.ID,
		Date:         res.Date,
		Lottery:      res.Lottery,
		QrData:       res.QrData,
		MerchantTin:  res.MerchantTIN,
		PosNo:        res.PosNo,
		Message:      res.Message,
	}

	if err := db.Create(&instance).Error; err != nil {
		fmt.Printf("Can't Save Ebarimt Data %v\n", err)
		return
	}

	for _, receipt := range res.Receipts {
		receiptInstance := EbarimtReceipt{
			EbarimtID:     instance.ID,
			BillID:        receipt.ID,
			TotalAmount:   receipt.TotalAmount,
			TotalVat:      receipt.TotalVat,
			TotalCityTax:  receipt.TotalCityTax,
			TaxType:       string(receipt.TaxType),
			MerchantTin:   receipt.MerchantTin,
			BankAccountNo: receipt.BankAccountNo,
		}

		if err := db.Create(&receiptInstance).Error; err != nil {
			fmt.Printf("Can't Save Ebarimt Receipt Data %v\n", err)
			return
		}

		// 🛠 ID-г хэвлэх, оноогдсон эсэхийг шалгах
		fmt.Printf("Saved Receipt ID: %d\n", receiptInstance.ID)

		for _, item := range receipt.Items {
			itemInstance := EbarimtReceiptItem{
				Name:               item.Name,
				BarCode:            item.BarCode,
				BarCodeType:        string(item.BarCodeType),
				ClassificationCode: item.ClassificationCode,
				MeasureUnit:        item.MeasureUnit,
				TaxProductCode:     item.TaxProductCode,
				Qty:                item.Qty,
				UnitPrice:          item.UnitPrice,
				TotalAmount:        item.TotalAmount,
				TotalVat:           item.TotalVat,
				TotalCityTax:       item.TotalCityTax,
				TotalBonus:         item.TotalBonus,
				ReceiptID:          receiptInstance.ID,
			}

			if err := db.Create(&itemInstance).Error; err != nil {
				fmt.Printf("Can't Save Ebarimt Receipt Item Data %v\n", err)
				return
			}
		}
	}
}
