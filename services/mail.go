package ebarimt3SdkServices

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/smtp"

	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	models "github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"
)

type EmailInput struct {
	Email    string                 `json:"email"`
	Subtitle string                 `json:"subtitle"`
	From     string                 `json:"from"`
	User     string                 `json:"user"`
	Password string                 `json:"password"`
	SmtpHost string                 `json:"smtp_host"`
	SmtpPort string                 `json:"smtp_port"`
	Response models.ReceiptResponse `json:"response"`
}

type SendEmailBody struct {
	Date         string `json:"date"`
	TotalAmount  string `json:"total_amount"`
	TotalVat     string `json:"total_vat"`
	BillType     string `json:"bill_type"`
	BillID       string `json:"bill_id"`
	Lottery      string `json:"lottery"`
	Type         string `json:"type"`
	TotalCityTax string `json:"total_city_tax"`
	Image        string `json:"image"`
	Items        []struct {
		Name         string `json:"name"`
		Qty          string `json:"qty"`
		TotalAmount  string `json:"total_amount"`
		TotalVat     string `json:"total_vat"`
		TotalCityTax string `json:"total_city_tax"`
	} `json:"items"`
}

func SendMail(input EmailInput) error {
	if input.Response.ID == "" {
		return errors.New("response id is required")
	}
	// imageUrl, err := NewStorageService(input.StorageEndpoint, input.StorageAccessKey, input.StorageSecretKey).AttachImage(&input.Response)
	// if err != nil {
	// 	return err
	// }

	emailBody := SendEmailBody{
		Date:         input.Response.Date,
		TotalAmount:  utils.FloatToStr(math.Round(input.Response.TotalAmount*100) / 100),
		TotalVat:     utils.FloatToStr(math.Round(input.Response.TotalVat*100) / 100),
		TotalCityTax: utils.FloatToStr(math.Round(input.Response.TotalCityTax*100) / 100),
		BillID:       input.Response.ID,
		Lottery:      input.Response.Lottery,
		Image:        "cid:qr-code",
	}

	if input.Response.Type == constants.RECEIPT_B2B_RECEIPT {
		emailBody.BillType = "Байгууллага"
		emailBody.Type = input.Response.CustomerTIN
	} else {
		emailBody.BillType = "Хувь хүн"
	}

	// for _, receipt := range input.Response.Receipts {
	// 	for _, item := range receipt.Items {
	// 		emailBody.Items = append(emailBody.Items, struct {
	// 			Name         string `json:"name"`
	// 			Qty          string `json:"qty"`
	// 			TotalAmount  string `json:"total_amount"`
	// 			TotalVat     string `json:"total_vat"`
	// 			TotalCityTax string `json:"total_city_tax"`
	// 		}{
	// 			Name:         item.Name,
	// 			Qty:          utils.FloatToStr(item.Qty),
	// 			TotalAmount:  utils.FloatToStr(math.Round(item.TotalAmount*100) / 100),
	// 			TotalVat:     utils.FloatToStr(math.Round(item.TotalVat*100) / 100),
	// 			TotalCityTax: utils.FloatToStr(math.Round(item.TotalCityTax*100) / 100),
	// 		})
	// 	}
	// }

	// wd, err := os.Getwd()
	// if err != nil {
	// 	return err
	// }

	templatePath := "../files/mail/ebarimt.html"

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		fmt.Println("Ebarimt mail template error: ", err)
		return err
	}

	var htmlBuf bytes.Buffer
	if err := t.Execute(&htmlBuf, emailBody); err != nil {
		return err
	}

	htmlBody, boundary, err := utils.GenerateInlineQR(htmlBuf.String(), input.Response.QrData)
	if err != nil {
		return err
	}

	var fullEmail bytes.Buffer
	fullEmail.WriteString(fmt.Sprintf("From: %s\r\n", input.From))
	fullEmail.WriteString(fmt.Sprintf("To: %s\r\n", input.Email))
	fullEmail.WriteString("Subject: Төлбөрийн баримт\r\n")
	fullEmail.WriteString("MIME-Version: 1.0\r\n")
	fullEmail.WriteString(fmt.Sprintf("Content-Type: multipart/related; boundary=%s\r\n", boundary))
	fullEmail.WriteString("\r\n")

	fullEmail.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	fullEmail.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	fullEmail.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	fullEmail.Write(htmlBuf.Bytes()) // original HTML template content
	fullEmail.WriteString("\r\n")
	fullEmail.Write(htmlBody) // or attach image parts here

	auth := smtp.PlainAuth("", input.User, input.Password, input.SmtpHost)

	// Sending email.
	err = smtp.SendMail(input.SmtpHost+":"+input.SmtpPort, auth, input.From, []string{input.Email}, fullEmail.Bytes())
	if err != nil {
		return err
	}

	return nil

}
