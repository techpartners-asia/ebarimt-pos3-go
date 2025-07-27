package ebarimt3SdkServices

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/smtp"
	"os"
	"path/filepath"

	"github.com/techpartners-asia/ebarimt-pos3-go/constants"
	models "github.com/techpartners-asia/ebarimt-pos3-go/structs"
	"github.com/techpartners-asia/ebarimt-pos3-go/utils"
)

type EmailInput struct {
	Email            string                 `json:"email"`
	Subtitle         string                 `json:"subtitle"`
	From             string                 `json:"from"`
	Password         string                 `json:"password"`
	SmtpHost         string                 `json:"smtp_host"`
	SmtpPort         string                 `json:"smtp_port"`
	Response         models.ReceiptResponse `json:"response"`
	StorageEndpoint  string                 `json:"storage_endpoint"`
	StorageAccessKey string                 `json:"storage_access_key"`
	StorageSecretKey string                 `json:"storage_secret_key"`
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
	imageUrl, err := NewStorageService(input.StorageEndpoint, input.StorageAccessKey, input.StorageSecretKey).AttachImage(&input.Response)
	if err != nil {
		return err
	}

	emailBody := SendEmailBody{
		Date:         input.Response.Date,
		TotalAmount:  utils.FloatToStr(math.Round(input.Response.TotalAmount*100) / 100),
		TotalVat:     utils.FloatToStr(math.Round(input.Response.TotalVat*100) / 100),
		TotalCityTax: utils.FloatToStr(math.Round(input.Response.TotalCityTax*100) / 100),
		BillID:       input.Response.ID,
		Lottery:      input.Response.Lottery,
		Image:        imageUrl,
	}

	if input.Response.Type == constants.RECEIPT_B2B_RECEIPT {
		emailBody.BillType = "Байгууллага"
		emailBody.Type = input.Response.CustomerTIN
	} else {
		emailBody.BillType = "Хувь хүн"
	}

	for _, receipt := range input.Response.Receipts {
		for _, item := range receipt.Items {
			emailBody.Items = append(emailBody.Items, struct {
				Name         string `json:"name"`
				Qty          string `json:"qty"`
				TotalAmount  string `json:"total_amount"`
				TotalVat     string `json:"total_vat"`
				TotalCityTax string `json:"total_city_tax"`
			}{
				Name:         item.Name,
				Qty:          utils.FloatToStr(item.Qty),
				TotalAmount:  utils.FloatToStr(math.Round(item.TotalAmount*100) / 100),
				TotalVat:     utils.FloatToStr(math.Round(item.TotalVat*100) / 100),
				TotalCityTax: utils.FloatToStr(math.Round(item.TotalCityTax*100) / 100),
			})
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	templatePath := filepath.Join(wd, "files/mail/ebarimt.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		fmt.Println("Ebarimt mail template error: ", err)
		return err
	}

	var body bytes.Buffer

	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	body.Write([]byte(fmt.Sprintf("Subject: %s \n%s\n\n", "Төлбөрийн баримт", mimeHeaders)))

	t.Execute(&body, emailBody)
	auth := smtp.PlainAuth("", input.From, input.Password, input.SmtpHost)

	// Sending email.
	err = smtp.SendMail(input.SmtpHost+":"+input.SmtpPort, auth, input.From, []string{input.Email}, body.Bytes())
	if err != nil {
		return err
	}

	return nil

}
