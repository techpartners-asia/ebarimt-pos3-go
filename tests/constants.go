package tests

import ebarimt3 "github.com/techpartners-asia/ebarimt-pos3-go"

const (
	OrgCode      = ""
	BranchNo     = "2"
	DistrictCode = "3420"
)

func NewSdk() *ebarimt3.EbarimtClient {
	sdk := ebarimt3.New(ebarimt3.Input{
		Endpoint:    "http://103.50.205.106:7080",
		MerchantTin: "37900846788",
		PosNo:       "101317341",
		DB:          nil,
		IsDev:       true,
	})

	return sdk
}
