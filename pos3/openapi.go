package pos3

import (
	"encoding/json"

	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
)

// Цахим төлбөрийн баримтын систем /PosApi/-ээс үүсгэж буй төлбөрийн баримтын үйл ажиллагаа явуулж буй байршлын мэдээллийг “districtCode” гэсэн баганад бөглөн илгээдэг ба дээрх талбарт бөглөн, илгээх байршлын татварын алба, дэд албаны кодын жагсаалтын мэдээллийг энэхүү сервисээс авах боломжтой.
//
// Жишээ нь: Номин холдинг ХХК-ийн Архангай аймаг дахь салбараас үүсгэсэн баримтын “districtCode”-г 01-гэж бөглөн илгээнэ.
//
// Ebarimt.mn API call: GET /api/info/check/getBranchInfo
//
// See https://developer.itc.gov.mn/docs/ebarimt-api/vxfs8o0ezfnsa-tatvaryn-alba-ded-albany-nerijn-kod-zhagsaaltyn-servis
func (p *pos3) GetBranchInfo() (structs.GetBranchInfoResponse, error) {
	response, err := p.httpRequest(nil, GetBranchInfoAPI, "", nil)
	if err != nil {
		return structs.GetBranchInfoResponse{}, err
	}
	var resp structs.GetBranchInfoResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) GetTinInfo(regNo string) (structs.GetTinInfoResponse, error) {
	response, err := p.httpRequest(nil, GetTinInfoAPI, regNo, nil)
	if err != nil {
		return structs.GetTinInfoResponse{}, err
	}
	var resp structs.GetTinInfoResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) GetInfo(regNo string) (structs.GetInfoResponse, error) {
	response, err := p.httpRequest(nil, GetInfoAPI, regNo, nil)
	if err != nil {
		return structs.GetInfoResponse{}, err
	}
	var resp structs.GetInfoResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) GetSalesTotalData(body structs.GetSalesTotalDataRequest) (structs.GetSalesTotalDataResponse, error) {
	var headers []CustomHeader
	header := CustomHeader{
		Name:  "X-API-KEY",
		Value: p.apiKey,
	}
	headers = append(headers, header)
	response, err := p.httpRequest(body, GetSalesTotalAPI, "", headers)
	if err != nil {
		return structs.GetSalesTotalDataResponse{}, err
	}
	var resp structs.GetSalesTotalDataResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) GetSalesListERP(body structs.GetSalesListERPRequest) (structs.GetSalesTotalDataResponse, error) {
	var headers []CustomHeader
	header := CustomHeader{
		Name:  "X-API-KEY",
		Value: p.apiKey,
	}
	headers = append(headers, header)
	response, err := p.httpRequest(body, GetSalesListERPAPI, "", headers)
	if err != nil {
		return structs.GetSalesTotalDataResponse{}, err
	}
	var resp structs.GetSalesTotalDataResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}

func (p *pos3) SaveOprMerchants(body structs.SaveOprMerchantsRequest) (structs.SaveOprMerchantsResponse, error) {
	response, err := p.httpRequest(body, GetSalesListERPAPI, "", nil)
	if err != nil {
		return structs.SaveOprMerchantsResponse{}, err
	}
	var resp structs.SaveOprMerchantsResponse
	json.Unmarshal(response, &resp)
	return resp, nil
}
