package pos3

import (
	"encoding/json"

	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
)

func (p *pos3) ConsumerInfo(regNo string) (structs.ConsumerInfoResponse, error) {
	response, err := p.httpRequest(nil, ConsumerInfoAPI, regNo, nil)
	if err != nil {
		return structs.ConsumerInfoResponse{}, err
	}
	var resp structs.ConsumerInfoResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) GetProfile(body structs.GetProfileRequest) (structs.GetProfileResponse, error) {
	response, err := p.httpRequest(body, GetProfileAPI, "", nil)
	if err != nil {
		return structs.GetProfileResponse{}, err
	}
	var resp structs.GetProfileResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) ApproveQr(body structs.ApproveQrRequest) (structs.ApproveQrResponse, error) {
	response, err := p.httpRequest(body, ApproveQrAPI, "", nil)
	if err != nil {
		return structs.ApproveQrResponse{}, err
	}
	var resp structs.ApproveQrResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) ForiegnerPassportInfo(fNumber, passportNo string) (structs.ForiegnerInfoResponse, error) {
	response, err := p.httpRequest(nil, ForiegnerPassportInfoAPI, passportNo+"/"+fNumber, nil)
	if err != nil {
		return structs.ForiegnerInfoResponse{}, err
	}
	var resp structs.ForiegnerInfoResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) ForiegnerCustomerNoInfo(loginName string) (structs.ForiegnerInfoResponse, error) {
	response, err := p.httpRequest(nil, ForiegnerCustomerNoInfoAPI, loginName, nil)
	if err != nil {
		return structs.ForiegnerInfoResponse{}, err
	}
	var resp structs.ForiegnerInfoResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) ForiegnerInfoRegister(passportNo string, body structs.ForiegnerInfoRequest) (structs.ForiegnerInfoResponse, error) {
	response, err := p.httpRequest(body, ForiegnerCustomerNoInfoAPI, passportNo, nil)
	if err != nil {
		return structs.ForiegnerInfoResponse{}, err
	}
	var resp structs.ForiegnerInfoResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}
