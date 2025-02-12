package structs

import "github.com/techpartners-asia/ebarimt-pos3-go/constants"

// region Нэгдсэн нэвтрэлт
type (

	// Token авах хүсэлт
	TokenRequest struct {
		GrantType string `json:"grant_type"` // Example: password
		Username  string `json:"username"`   // Example: easy-register-test
		Password  string `json:"password"`   // Example: Test@123
		ClientID  string `json:"client_id"`  // Example: vatps
	}

	// Token авах хариу
	TokenResponse struct {
		AccessToken      string `json:"access_token"` // Use this token
		ExpiresIn        string `json:"expires_in"`
		RefreshToken     string `json:"refresh_token"`
		RefreshExpiresIn string `json:"refresh_expires_in"`
		TokenType        string `json:"token_type"`
		NotBeforePolicy  string `json:"not-before-policy"`
		SessionState     string `json:"session_state"`
		Scope            string `json:"scope"`
	}

	// Алдааны хариу
	ErrorResponse struct {
		Path             string `json:"path"`
		Status           int    `json:"status"`
		Message          string `json:"message"`
		Timestamp        string `json:"timestamp"`
		RequestID        string `json:"requestId"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
)

//endregion

// region Цахим төлбөрийн баритм API холболт
type (

	// Татварын алба, дэд албаны нэрийн, код жагсаалтын сервисийн хариу
	GetBranchInfoResponse struct {
		Msg    string              `json:"msg"`    // Сервисийн хариу
		Status int                 `json:"status"` // Service ийн төлөв
		Code   string              `json:"code"`   // Код
		Data   []GetBranchInfoData `json:"data"`   // Өгөгдөл
	}

	// Татварын алба, дэд албаны нэрийн, код жагсаалт мэдээлэл
	GetBranchInfoData struct {
		BranchCode    string `json:"branchCode"`    // Салбарын код
		BranchName    string `json:"branchName"`    // Салбарын нэр
		SubBranchCode string `json:"subBranchCode"` // Дэд салбарын код
		SubBranchName string `json:"subBranchName"` // Дэд салбарын нэр
	}

	// Татвар төлөгчийн дугаар лавлах сервис /ТIN, Civil_id/ хариу
	GetTinInfoResponse struct {
		Msg    string `json:"msg"`    // Сервисийн хариу
		Status int    `json:"status"` // Service ийн төлөв
		Code   string `json:"code"`   // Код
		Data   string `json:"data"`   // ТИН дугаар
	}

	// Татвар төлөгчийн бүртгэлийн мэдээлэл лавлах сервис хариу
	GetInfoResponse struct {
		Msg    string      `json:"msg"`    // Сервисийн хариу
		Status int         `json:"status"` // Service ийн төлөв
		Code   string      `json:"code"`   // Код
		Data   GetInfoData `json:"data"`   // Өгөгдөл
	}

	// Татвар төлөгчийн бүртгэлийн мэдээлэл
	GetInfoData struct {
		Name                   string `json:"name"`                   // Нэр
		FreeProject            bool   `json:"freeProject"`            // НӨАТ-аас чөлөөлөгдөх төсөл хөтөлбөр эсэх
		CityPayer              bool   `json:"cityPayer"`              // НХАТ суутгагч эсэх
		VatPayer               bool   `json:"vatPayer"`               // НӨАТ суутган төлөгч эсэх
		Found                  bool   `json:"found"`                  // Татвар төлөгчөөр бүртгэлтэй эсэх
		VatPayerRegisteredDate string `json:"vatpayerRegisteredDate"` // НӨАТ суутган төлөгчөөр бүртгүүлсэн огноо
	}

	// Борлуулалтын задаргааны мэдээлэл татах сервисийн хүсэлт
	GetSalesTotalDataRequest struct {
		Year       string              `json:"year"`       // Задаргаа авч буй баримтын он
		Month      string              `json:"month"`      // Задаргаа авч буй баримтын сар
		Day        string              `json:"day"`        // Задаргаа авч буй баримтын өдөр /Заавал биш/
		Status     constants.SalesData `json:"status"`     // Задаргаа авч буй баримтын төрөл
		StartCount int                 `json:"startCount"` // Тухайн баримтын эхлэх тоо
		EndCount   int                 `json:"endCount"`   // Тухайн баримтын дуусах тоо
	}

	// Борлуулалтын задаргааны мэдээлэл татах сервисийн хариу
	GetSalesTotalDataResponse struct {
		Msg    string            `json:"msg"`    // Сервисийн хариу
		Status int               `json:"status"` // Service ийн төлөв
		Code   string            `json:"code"`   // Код
		Data   GetSalesTotalData `json:"data"`   // Өгөгдөл
	}
	GetSalesTotalData struct {
		Content   []GetSalesData `json:"content"`
		PageModel PageModel      `json:"pageModel"`
	}

	// Борлуулалтын задаргааны мэдээлэл
	GetSalesData struct {
		PosSid       string  `json:"posSid"`       // Посын системийн дугаар
		PosRno       string  `json:"posRno"`       // Төлбөрийн баримтын ДДТД
		PosRdate     string  `json:"posRdate"`     // Төлбөрийн баримтын огноо
		PosRamt      float64 `json:"posRamt"`      // Төлбөрийн баримтын нийт дүн
		CityTax      float64 `json:"cityTax"`      // Нийслэл хотын албан татвар
		PosVamt      float64 `json:"posVamt"`      // Нэмэгдсэн өртгийн албан татвар
		NetAmt       float64 `json:"netAmt"`       // Төлбөрийн баримтын цэвэр дүн
		FromType     string  `json:"fromType"`     // Хаанаас үүсгэсэн /POS, ebarimt/
		CsmrRegNo    string  `json:"csmrRegNo"`    // Худалдан авагчийн регистр
		CsmrName     string  `json:"csmrName"`     // Худалдан авагчийн нэр
		PosNo        string  `json:"posNo"`        // Посын дугаар
		OperatorName string  `json:"operatorName"` // Хэрэглэгчийн систем нийлүүлэгчийн нэр
		DistrictCode string  `json:"districtCode"` // Төлбөрийн баримтын байршлын нэр
	}

	// Борлуулалтын задаргааны хуудасны мэдээлэл
	PageModel struct {
		TotalElements int `json:"totalElements"` // Нийт баримтын тоо
	}
	// Толгой татвар төлөгч өөрийн охин компанийн худалдан авалт татах сервисийн хүсэлт
	GetSalesListERPRequest struct {
		Pin       string   `json:"Pin"`       // Толгой компанийн регистрийн дугаар
		StartDate string   `json:"StartDate"` // Баримт татах эхлэх огноо
		SubPin    []string `json:"subPin"`    // Охин компанийн регистрийн дугаар. Хэрэв регистр оруулахгүй бол толгой компанийн мэдээлэл татагдах
		EndDate   int      `json:"EndDate"`   // Баримт татах дуусах огноо
	}

	// Толгой татвар төлөгч өөрийн охин компанийн худалдан авалт татах сервисийн хариу
	GEtSalesListERPResponse struct {
		Msg    string              `json:"msg"`    // Сервисийн хариу
		Status int                 `json:"status"` // Service ийн төлөв
		Code   string              `json:"code"`   // Код
		Data   GEtSalesListERPData `json:"data"`   // Өгөгдөл
	}
	GEtSalesListERPData struct {
		StartDate           string            `json:"startDate"`           // эхлэх огноо
		EndDate             int               `json:"endDate"`             // дуусах огноо
		RegNo               string            `json:"regNo"`               // Толгой компанийн регистр
		RecieptBuyModelList []RecieptBuyModel `json:"receiptBuyModelList"` // Худалдан авалтын баримтын жагсаалт
	}

	// Худалдан авалтын баримтын мэдээлэл
	RecieptBuyModel struct {
		PrPosRno      string  `json:"prPosRno"`      // Төлбөрийн баримтын ДДТД
		Name          string  `json:"name"`          // Борлуулагчийн нэр
		RegNo         string  `json:"regNo"`         // Борлуулагчийн регистр
		BuyerRegNo    string  `json:"buyerRegNo"`    // Худалдан авагчийн регистр
		Date          string  `json:"date"`          // Баримт хэвлэсэн огноо
		AmountVAT     float64 `json:"amountVat"`     // НӨАТ
		AmountCityTax float64 `json:"amountCityTax"` // НХАТ
		AmountTotal   float64 `json:"amountTotal"`   // Төлбөрийн баримтын нийт дүн
		AmountNet     float64 `json:"amountNet"`     // Төлбөрийн баримтын цэвэр дүн
		FromType      string  `json:"fromType"`      // Хаанаас баримт хэвлэсэн /INVOICE, POS API/
	}

	// Оператороос мерчант бүртгэх хүсэлт илгээх сервисийх хүсэлт
	SaveOprMerchantsRequest struct {
		OprRegNo string   `json:"oprRegNo"` // Оператор компани буюу Хэрэглэгчийн систем нийлүүлэгчийн регистр
		PosNo    string   `json:"posNo"`    // Пос-ын дугаар
		List     []string `json:"list"`     // Бүртгэх мерчантын жагсаалт
	}

	// Оператороос мерчант бүртгэх хүсэлт илгээх сервисийх хариу
	SaveOprMerchantsResponse struct {
		Msg    string        `json:"msg"`    // Сервисийн хариу
		Status int           `json:"status"` // Service ийн төлөв
		Code   string        `json:"code"`   // Код
		Data   []interface{} `json:"data"`   // Өгөгдөл
	}
)

// endregion

// region Хялбар бүртгэлийн API холболт
type (

	// Иргэний мэдээлэл регистрийн дугаараар лавлах сервисийн хариу
	ConsumerInfoResponse struct {
		RegNo      string `json:"regNo"`      //Иргэний регистрийн дугаар
		LoginName  string `json:"loginName"`  // Ebarimt-н нэвртэх нэр буюу 8 оронтой хэрэглэгчийн код
		GiveName   string `json:"giveName"`   // Хэрэглэгчийн нэр
		FamilyName string `json:"familyName"` // Хэрэглэгчийн овог
	}

	// Иргэний мэдээллийг утасны дугаараар болон хэрэглэгчийн дугаараар лавлах сервисийн хүсэлт
	GetProfileRequest struct {
		PhoneNum   string `json:"phoneNum"`   // Иргэний утасны дугаар /Хэрэглэгчийн дугаар талбарт өгөдөл оруулсан бол заавал биш/
		CustomerNo string `json:"customerNo"` // Хэрэглэгчийн дугаарр./Иргэний утасны дугаар талбарт өгөдөл оруулсан бол заавал биш/
	}

	// Иргэний мэдээллийг утасны дугаараар болон хэрэглэгчийн дугаараар лавлах сервисийн хариу
	GetProfileResponse struct {
		Msg       string `json:"msg"`       // Сервисийн хариу
		Status    int    `json:"status"`    // Service ийн төлөв
		Code      string `json:"code"`      // Сервисийн хариу
		Email     string `json:"email"`     // Хэрэглэгчийн цахим шуудан хаяг
		LoginName string `json:"loginName"` // Ebarimt-н нэвртэх нэр буюу 8 оронтой хэрэглэгчийн код.
	}

	// Төлбөрийн баримт баталгаажуулах сервисийн хүсэлт
	ApproveQrRequest struct {
		CustomerNo string `json:"customerNo"`
		QrData     string `json:"qrData"`
	}

	// Төлбөрийн баримт баталгаажуулах сервисийн хариу
	ApproveQrResponse struct {
		Msg    string `json:"msg"`    // Сервисийн хариу
		Status int    `json:"status"` // Service ийн төлөв
		Code   string `json:"code"`   // Сервисийн хариу
	}

	// Гадаад жуулчны мэдээллийг лавлах сервисийн хариу
	ForiegnerInfoResponse struct {
		RegNo      string `json:"regNo"`      // Гадаад иргэний регистрийн дугаар.
		LoginName  string `json:"loginName"`  // Ebarimt-н нэвтрэх нэр буюу 8 оронтой хэрэглэгчийн код.
		GiveName   string `json:"givenName"`  // Хэрэглэгчийн нэр.
		FamilyName string `json:"familyName"` // Хэрэглэгчийн овог.
		Refund     string `json:"refund"`     // Буцаан олголт авах эсэх.
		Country    string `json:"country"`    // Улсын нэр.
		PassportNo string `json:"passportNo"` // Гадаад паспортын дугаар.
		FNumber    string `json:"fNumber"`    // F регистрийн дугаар.
	}

	// Гадаад жуулчны мэдээллийг E-barimt-н системд бүртгэх сервисийн хүсэлт
	ForiegnerInfoRequest struct {
		Email string `json:"email"`
	}
)

//endregion

// region Pos хүсэлт
type (

	// Баримтын хүсэлт
	ReceiptRequest struct {
		TotalAmount  float64               `json:"totalAmount"`  // Багц баримтын гүйлгээний нийт дүн (Бүх төрлийн татвар шингэсэн дүн)
		TotalVat     float64               `json:"totalVat"`     // Багц баримтын НӨАТ-н нийт дүн
		TotalCityTax float64               `json:"totalCityTax"` // Багц баримтын НХАТ-н нийт дүн
		BranchNo     string                `json:"branchNo"`     // Салбарын дугаар
		DistrictCode string                `json:"districtCode"` // Баримт хэвлэсэн орон нутгийн код /4 оронтой бүхэл тоо/
		MerchantTin  string                `json:"merchantTin"`  // Багц баримт олгогчийн ТТД (11 эсвэл 14 оронтой бүхэл тоо )
		PosNo        string                `json:"posNo"`        // Тухайн байгууллагын дотоод кассын дугаар
		CustomerTin  string                `json:"customerTin"`  // Худалдан авагчийн ТТД
		ConsumerNo   string                `json:"consumerNo"`   // Худалдан авагч иргэний ebarimt-н бүртгэлийн дугаар
		Type         constants.ReceiptType `json:"type"`         // Баримтын төрөл
		InActiveID   string                `json:"inActiveId"`   // Засварлах баримтын ДДТД (33 орон бүхий тоон утга)
		InvoiceID    string                `json:"invoiceId"`    // Тухайн төлбөрийн баримтын харгалзах нэхэмжлэхийн ДДТД (33 орон бүхий тоон утга)
		ReportMonth  interface{}           `json:"reportMonth"`  // Баримт харьяалагдах тайлант сар ("yyyy-MM-dd" форматтай огноо)
		Data         interface{}           `json:"data"`         // Багц төлбөрийн баримтын нэмэлт өгөгдөл JSON
		Receipts     []Receipt             `json:"receipts"`     // Дэд төлбөрийн баримтууд
		Payments     []Payment             `json:"payments"`     // Төлбөрийн хэлбэр
	}

	// Дэд төлбөрийн баримтын мэдээлэл
	Receipt struct {
		ID            string            `json:"id"`            // Дэд төлбөрийн баримтын ДДТД
		TotalAmount   float64           `json:"totalAmount"`   // Дэд төлбөрийн баримтын гүйлгээний нийт дүн (Бүх төрлийн татвар шингэсэн дүн)
		TotalVat      float64           `json:"totalVat"`      // Дэд төлбөрийн баримтын НӨАТ-н нийт дүн
		TotalCityTax  float64           `json:"totalCityTax"`  // Дэд төлбөрийн баримтын НХАТ-н нийт дүн
		TaxType       constants.TaxType `json:"taxType"`       // Татварын төрөл
		MerchantTin   string            `json:"merchantTin"`   // Борлуулагчийн ТТД
		BankAccountNo string            `json:"bankAccountNo"` // Нэхэмжлэхийн банкны дансны дугаар
		Data          interface{}       `json:"data"`          // Дэд төлбөрийн баримтын нэмэлт өгөгдөл.
		Items         []Item            `json:"items"`         // Борлуулсан бүтээгдэхүүн, үйлчилгээний жагсаалт
	}

	// Борлуулсан бүтээгдэхүүн, үйлчилгээний мэдээлэл
	Item struct {
		Name               string                `json:"name"`               // Бүтээгдэхүүн, үйлчилгээний нэр
		BarCode            string                `json:"barCode"`            // Бүтээгдэхүүний зураасан код
		BarCodeType        constants.BarcodeType `json:"barCodeType"`        // Зураасан кодын төрөл
		ClassificationCode string                `json:"classificationCode"` // Бүтээгдэхүүн, үйлчилгээний ангиллын код (7 оронтой бүхэл тоо)
		TaxProductCode     string                `json:"taxProductCode"`     // taxType талбарын утга нь VAT_FREE, VAT_ZERO үед татварын харгалзах 3 оронтой тоон кодыг оруулана.
		MeasureUnit        string                `json:"measureUnit"`        // Хэмжих нэгж
		Qty                float64               `json:"qty"`                // Борлуулсан тоо, хэмжээ
		UnitPrice          float64               `json:"unitPrice"`          // Нэгж үнэ (Бүх төрлийн татвар шингэсэн дүн)
		TotalBonus         float64               `json:"totalBonus"`
		TotalVat           float64               `json:"totalVat"`     // Бүтээгдэхүүн, үйлчилгээний НӨАТ-н нийт дүн
		TotalCityTax       float64               `json:"totalCityTax"` // Бүтээгдэхүүн, үйлчилгээний НХАТ-н нийт дүн
		TotalAmount        float64               `json:"totalAmount"`  // Бүтээгдэхүүн, үйлчилгээний гүйлгээний нийт дүн (Бүх төрлийн татвар шингэсэн дүн)
		Data               interface{}           `json:"data"`         // Бүтээгдэхүүн, үйлчилгээний нэмэлт өгөгдөл
	}

	// Төлбөрийн хэлбэрийн мэдээлэл
	Payment struct {
		Code         constants.PaymentCode   `json:"code"`         // Төлбөрийн хэлбэрийн код
		ExchangeCode string                  `json:"exchangeCode"` // Төлбөр хийж гүйцэтгэх гуравдагч системийн код
		Status       constants.PaymentStatus `json:"status"`       // Төлбөрийн хэлбэрийн төлөв
		PaidAmount   float64                 `json:"paidAmount"`   // Нийт төлсөн дүн
		Data         interface{}             `json:"data"`         // Төлбөрийн нэмэлт өгөгдөл
	}

	// Баримтын хариу
	ReceiptResponse struct {
		ID           string                `json:"id"`           // Багц төлбөрийн баримтын ДДТД
		Version      string                `json:"version"`      // Version
		TotalAmount  float64               `json:"totalAmount"`  // Багц баримтын гүйлгээний нийт дүн (Бүх төрлийн татвар шингэсэн дүн)
		TotalVat     float64               `json:"totalVat"`     // Багц баримтын НӨАТ-н нийт дүн
		TotalCityTax float64               `json:"totalCityTax"` // Багц баримтын НХАТ-н нийт дүн
		BranchNo     string                `json:"branchNo"`     // Салбарын дугаар
		DistrictCode string                `json:"districtCode"` // Баримт хэвлэсэн орон нутгийн код /4 оронтой бүхэл тоо/
		MerchantTIN  string                `json:"merchantTin"`  // Багц баримт олгогчийн ТТД (11 эсвэл 14 оронтой бүхэл тоо )
		PosNo        string                `json:"posNo"`        // Тухайн байгууллагын дотоод кассын дугаар
		CustomerTIN  string                `json:"customerTin"`  // Худалдан авагчийн ТТД
		ConsumerNo   string                `json:"consumerNo"`   // Худалдан авагч иргэний ebarimt-н бүртгэлийн дугаар
		Type         constants.ReceiptType `json:"type"`         // Баримтын төрөл
		InvoiceID    string                `json:"invoiceId"`    // Тухайн төлбөрийн баримтын харгалзах нэхэмжлэхийн ДДТД (33 орон бүхий тоон утга)
		Receipts     []Receipt             `json:"receipts"`     // Дэд төлбөрийн баримтууд
		PosID        float64               `json:"posId"`        // PosAPI-н системийн дугаар
		Status       constants.PosStatus   `json:"status"`       // Баримтын төлөв
		Message      string                `json:"message"`      // Тайлбар утга
		QrData       string                `json:"qrData"`       // QR Code-н утга
		Lottery      string                `json:"lottery"`      // Сугалааны дугаар
		Date         string                `json:"date"`         // Баримт хэвэлсэн огноо ("yyyy-MM-dd HH:mm:ss" форматтай огноо)
		Easy         bool                  `json:"easy"`         // Хялбар бүртгэл хийгдсэн эсэх /true - хялбар бүртгэлд бүртгэсэн /false - хялбар бүртгэлд бүртгээгүй
	}

	// Дэд төлбөрийн баримтын мэдээлэл
	// ReceiptData struct {
	// 	ID            string `json:"id"`            // Дэд баримтын ДДТД
	// 	BankAccountID int    `json:"bankAccountId"` // ААН, Иргэний системд бүртгэсэн банкны дансны ID
	// }

	// PosAPI-руу төлбөрийн баримт буцаах сервисийн хүсэлт
	ReceiptDeleteRequest struct {
		ID   string `json:"id"`   // Багц төлбөрийн баримтын ДДТД (33 оронтой тоо)
		Date string `json:"date"` // Баримт хэвэлсэн огноо ("yyyy-MM-dd HH:mm:ss" форматтай огноо)
	}

	// PosAPI-ын ажиллагааны мэдээлэл хүлээн авах сервисийн хариу
	InfoResponse struct {
		OperatorName  string         `json:"operatorName"`  // Оператор байгууллагын нэр
		OperatorTin   string         `json:"operatorTIN"`   // Оператор байгууллагын ТТД
		PosID         float64        `json:"posId"`         // PosAPI-н систем дэх бүртгэлийн Id
		PosNo         string         `json:"posNo"`         // PosAPI-н систем дэх бүртгэлийн дугаар.
		LastSendDate  string         `json:"lastSendDate"`  // Баримт илгээсэн огноо /Сүүлийн байдлаар/.
		LeftLotteries int            `json:"leftLotteries"` // Нийт үлдсэн сугалаа
		AppInfo       AppInfo        `json:"appInfo"`       // PosAPI-н ерөнхий мэдээлэл
		Merchants     []MerchantData `json:"merchants"`     // PosAPI-д бүртгэлтэй ААН-н жагсаалт
	}

	// PosAPI-н ерөнхий мэдээлэл
	AppInfo struct {
		ApplicationDir string `json:"applicationDir"` // Файл байршиж буй хавтас
		CurrentDir     string `json:"currentDir"`     // Файл байршиж буй хавтас
		Database       string `json:"database"`       // Баазын driver /Qt5 тохиргоогоор/
		DatabaseHost   string `json:"database-host"`  // Баазын хаяг /sqlite бол файлын зам/
		WorkDir        string `json:"workDir"`        // Ажиллаж буй хавтас
	}

	// PosAPI-д бүртгэлтэй ААН мэдээлэл
	MerchantData struct {
		Name      string         `json:"name"`      // ААН-нэр
		TIN       string         `json:"tin"`       // ААН-н ТТД
		Customers []CustomerData `json:"customers"` // Үйлчлүүлэгч ААН-н жагсаалт
	}

	// Үйлчлүүлэгч ААН мэдээлэл
	CustomerData struct {
		Name string `json:"name"` // ААН-нэр
		TIN  string `json:"tin"`  // ААН-н ТТД
	}

	// Банкны дансны мэдээлэл хүлээн авах сервисийн мэдээлэл
	BankAccountData struct {
		ID              string `json:"id"`              // ААН, Иргэний системд бүртгэсэн банкны дансны ID
		TIN             int    `json:"tin"`             // Данс эзэмшигч ААН, Иргэний ТТД
		BankAccountNo   string `json:"bankAccountNo"`   // Банкны дансны дугаар
		BankAccountName string `json:"bankAccountName"` // Банкны дансны нэр
		BankID          int    `json:"bankId"`          // Банкны ID
		BankName        string `json:"bankName"`        // Банкны нэр
	}

	// Амжилттай хариу
	Response struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Date    string `json:"date"`
	}
)

// endregion
