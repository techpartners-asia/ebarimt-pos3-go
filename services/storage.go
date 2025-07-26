package ebarimt3SdkServices

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/techpartners-asia/ebarimt-pos3-go/structs"
)

type StorageService struct {
	minioClient *minio.Client
}

func NewStorageService(endpoint, accessKey, secretKey string) *StorageService {

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil
	}

	return &StorageService{
		minioClient: minioClient,
	}

}

func (s *StorageService) AttachImage(res *structs.ReceiptResponse) (string, error) {

	qrCode, _ := qr.Encode(res.QrData, qr.L, qr.Auto)
	qrCode, _ = barcode.Scale(qrCode, 512, 512)
	buf := new(bytes.Buffer)
	png.Encode(buf, qrCode)
	var imageData = bytes.NewReader(buf.Bytes())

	uploadInfo, errUploadInfo := s.minioClient.PutObject(context.Background(), "ebarimt", fmt.Sprintf("%v-%v.png", res.ID, time.Now().Format("20060102150405")), imageData, imageData.Size(), minio.PutObjectOptions{ContentType: "image/png"})
	fmt.Println("uploadInfo :", uploadInfo, errUploadInfo)

	if errUploadInfo != nil {
		return "", errUploadInfo
	}

	return fmt.Sprintf("%s/%s/%s", s.minioClient.EndpointURL(), uploadInfo.Bucket, uploadInfo.Key), nil
}
