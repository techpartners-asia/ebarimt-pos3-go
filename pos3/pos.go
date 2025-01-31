package pos3

import (
	"encoding/json"

	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
)

func (p *pos3) ReceiptSend(body structs.ReceiptRequest) (structs.ReceiptResponse, error) {
	response, err := p.httpPosRequest(body, PosReceiptSendAPI, "", nil)
	if err != nil {
		return structs.ReceiptResponse{}, err
	}
	var resp structs.ReceiptResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) ReceiptDelete(body structs.ReceiptDeleteRequest) (structs.Response, error) {
	response, err := p.httpPosRequest(body, PosReceiptDeleteAPI, "", nil)
	if err != nil {
		return structs.Response{}, err
	}
	var resp structs.Response
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) SendData() (structs.Response, error) {
	response, err := p.httpPosRequest(nil, PosSendAPI, "", nil)
	if err != nil {
		return structs.Response{}, err
	}
	var resp structs.Response
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) Info() (structs.InfoResponse, error) {
	response, err := p.httpPosRequest(nil, PosInfoAPI, "", nil)
	if err != nil {
		return structs.InfoResponse{}, err
	}
	var resp structs.InfoResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) BankAccounts(tin string) ([]structs.BankAccountData, error) {
	response, err := p.httpPosRequest(nil, PosBankAccAPI, "tin="+tin, nil)
	if err != nil {
		return nil, err
	}
	var resp []structs.BankAccountData
	json.Unmarshal(response, &resp)
	return resp, nil
}
