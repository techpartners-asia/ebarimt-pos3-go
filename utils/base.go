package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strconv"
	"time"

	"github.com/skip2/go-qrcode"
)

// int64Pointer Get int64 pointer
func int64Pointer(i int64) *int64 {
	return &i
}

// StrToUint String to Uint parser
func StrToUint(value string) (uint, error) {
	u64, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	result := uint(u64)
	return result, nil
}

func AppendAsString(args ...interface{}) string {
	appendedStr := ""
	for _, arg := range args {
		appendedStr = appendedStr + fmt.Sprintf("%v", arg)
	}

	return appendedStr
}

func GetValidString(source interface{}) string {
	if source == nil {
		return ""
	} else {
		return source.(string)
	}
}

func GetValidFloat(source interface{}) float64 {
	if source == nil {
		return float64(0)
	} else {
		num, _ := strconv.ParseFloat(source.(string), 64)
		return num
	}
}

func Float64ToString(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func NumberPrecision(value float64) float64 {
	return float64(int(value*100)) / 100
	// return float64(int(value*100)) / 100
}

func GetVat(value float64) float64 {
	return (value / 110) * 100000 / 10000
}

func GetVatWithCityTax(value float64) float64 {
	step1 := (value / 112) * 100000 / 10000
	return step1
}

func GetCityTax(value float64) float64 {
	return (value / 112) * 100000 / 50000
}

func GetCityTaxWithoutVat(value float64) float64 {
	return (value / 102) * 100000 / 50000
}

func FloatToStr(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func GenerateInlineQR(htmlBody, qrData string) ([]byte, string, error) {
	// Generate QR PNG
	qrBytes, err := qrcode.Encode(qrData, qrcode.Medium, 256)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// HTML part
	htmlHeader := textproto.MIMEHeader{}
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	htmlPart, _ := writer.CreatePart(htmlHeader)
	htmlPart.Write([]byte(htmlBody))

	// Inline image part
	imgHeader := textproto.MIMEHeader{}
	imgHeader.Set("Content-Type", "image/png")
	imgHeader.Set("Content-Transfer-Encoding", "base64")
	imgHeader.Set("Content-ID", "<qr-code>")
	imgPart, _ := writer.CreatePart(imgHeader)
	imgPart.Write([]byte(base64.StdEncoding.EncodeToString(qrBytes)))

	writer.Close()

	return buf.Bytes(), writer.Boundary(), nil
}

func FormatDate(date string) string {
	parsedDate, err := time.Parse(date, "2006-01-02")
	if err != nil {
		return "-"
	}
	return parsedDate.Format("2006-01-02")
}
