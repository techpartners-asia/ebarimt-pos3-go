package constants

type SalesData int

const (
	SALES_DATA_ALL     SalesData = iota //  Нийт борлуулалтын барим
	SALES_DATA_B2B                      // Бизнес эрхлэгч хооронд үүсгэсэн баримт
	SALES_DATA_LOTTERY                  // Эцсийн хэрэглээ буюу сугалаатай баримт
	SALES_DATA_INVOICE                  // Нэхэмжлэх
	SALES_DATA_BATCH                    // Багцын толгой баримт
)

type PosStatus string

const (
	POS_STATUS_SUCCESS PosStatus = "SUCCESS" // Төлбөрийн баримтын мэдээлэл амжилттай үүссэн.
	POS_STATUS_ERROR   PosStatus = "ERROR"   // Төлбөрийн баримтын мэдээлэл үүсгэхэд алдаа гарсан.
	POS_STATUS_PAYMENT PosStatus = "PAYMENT" // Төлбөрийн баримтын мэдээлэл үүсгэхэд төлбөрийн мэдээлэл дутуу.
)

type TaxType string

const (
	TAX_VAT_ABLE TaxType = "VAT_ABLE" // НӨАТ тооцох бүтээгдэхүүн, үйлчилгээ
	TAX_VAT_FREE TaxType = "VAT_FREE" // НӨАТ-аас чөлөөлөгдөх бүтээгдэхүүн, үйлчилгээ
	TAX_VAT_ZERO TaxType = "VAT_ZERO" // НӨАТ-н 0 хувь тооцох бүтээгдэхүүн, үйлчилгээ
	TAX_NO_VAT   TaxType = "NOT_VAT"  // Монгол улсын хилийн гадна борлуулсан бүтээгдэхүүн үйлчилгээ
)

type ReceiptType string

const (
	RECEIPT_B2C_RECEIPT ReceiptType = "B2C_RECEIPT"
	RECEIPT_B2B_RECEIPT ReceiptType = "B2B_RECEIPT"
	RECEIPT_B2C_INVOICE ReceiptType = "B2C_INVOICE"
	RECEIPT_B2B_INVOICE ReceiptType = "B2B_INVOICE"
)

type PaymentCode string

const (
	PAYMENT_CASH PaymentCode = "CASH"         // Бэлнээр
	PAYMENT_CARD PaymentCode = "PAYMENT_CARD" // Төлбөрийн карт
)

type PaymentStatus string

const (
	STATUS_PAID     PaymentStatus = "PAID"     // Төлбөр амжилттай хийгдсэнийг тодорхойлоно
	STATUS_PAY      PaymentStatus = "PAY"      // Төлбөрийн мэдээллийг “Баримтын мэдээлэл солилцох сервис”-г ашиглан гүйцэтгэнэ.
	STATUS_REVERSED PaymentStatus = "REVERSED" // Төлбөр буцаагдсан
	STATUS_ERROR    PaymentStatus = "ERROR"    // Төлөлт амжилтгүй болсон
)

type BarcodeType string

const (
	BARCODE_UNDEFINED BarcodeType = "UNDEFINED"
	BARCODE_GS1       BarcodeType = "GS1"
	BARCODE_ISBN      BarcodeType = "ISBN"
)
