package tests

import ebarimt3 "github.com/techpartners-asia/ebarimt-pos3-go/client"

const (
	OrgCode      = ""
	BranchNo     = "2"
	DistrictCode = "3420"
)

func NewSdk() *ebarimt3.EbarimtClient {
	sdk := ebarimt3.New(ebarimt3.Input{
		Endpoint:    "http://103.50.205.106:7080",
		MerchantTin: "87001066048",
		PosNo:       "10005608",
		DB:          nil,
	})

	return sdk
}
