package utils

type API struct {
	Url    string `json:"url"`
	Method string `json:"method"`
	IsAuth bool   `json:"isAuth"`
	DevUrl string `json:"dev_url"`
}

const (
	TimeFormatYYYYMMDDHHMMSS = "20060102150405"
	TimeFormatYYYYMMDD       = "20060102"
	HttpContentType          = "application/x-www-form-urlencoded"
	HttpAcceptPublic         = "application/javascript, application/xml, application/json"
	HttpAcceptPrivate        = "application/json"
	XForm                    = "application/x-www-form-urlencoded"
	XmlContent               = "application/xml"
)
