package ebarimt3

import (
	"github.com/techpartners-asia/ebarimt-pos3-go/pos3"
)

type (
	EbarimtClient struct {
		pos3.Pos3
	}
	Input struct {
		Endpoint string
		PosNo string
		MerchantTin string
	}
)

func Init(input Input) *EbarimtClient {
	posv3 := pos3.New(pos3.ConnectionInput{
		PosEndpoint: input.Endpoint,
		PosNo:       input.PosNo,
		MerchantTin: input.MerchantTin,
	})
	return &EbarimtClient{
		posv3,
	}
}
