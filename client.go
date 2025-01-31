package ebarimt3

import (
	"github.com/spf13/viper"
	"github.com/techpartners-asia/ebarimt-pos3-go/pos3"
)

type EbarimtClient struct {
	pos3.Pos3
}

func Init() *EbarimtClient {
	posv3 := pos3.New(pos3.ConnectionInput{
		PosEndpoint: viper.GetString("EBARIMT_ENDPOINT"),
		PosNo:       viper.GetString("EBARIMT_POS_NO"),
		MerchantTin: viper.GetString("EBARIMT_MERCHANT_TIN"),
	})
	return &EbarimtClient{
		posv3,
	}
}
