# 🧾 ebarimt-pos3-go

[![Go Report Card](https://goreportcard.com/badge/github.com/techpartners-asia/ebarimt-pos3-go)](https://goreportcard.com/report/github.com/techpartners-asia/ebarimt-pos3-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.23.5-blue.svg)](https://golang.org)
[![GORM](https://img.shields.io/badge/GORM-v1.25.12-green.svg)](https://gorm.io)

> Ebarimt POS 3.0 Golang Implementation SDK - A comprehensive Go library for integrating with the Ebarimt POS 3.0 system.

> **v2 is available** (`github.com/techpartners-asia/ebarimt-pos3-go/v2`, in [`v2/`](v2/)): a rewrite from the PosAPI 3.0 docs with exact money, a pure receipt builder and strict errors. v1 below is unchanged. See [v2/MIGRATION.md](v2/MIGRATION.md).

## ✨ Features

- 🚀 Complete POS 3.0 API implementation
- 🛡️ Type-safe Go structures for all API requests and responses
- ⚡ Comprehensive error handling
- 🔐 Built-in authentication and security features
- ✅ Extensive test coverage
- 💾 GORM integration for database operations

## 📦 Installation

```bash
go get github.com/techpartners-asia/ebarimt-pos3-go
```

## 🚀 Quick Start

```go
package main

import (
    "github.com/techpartners-asia/ebarimt-pos3-go"
    "gorm.io/gorm"
)

func main() {
    // Initialize the client with required parameters
    client := ebarimtv3.New(ebarimtv3.Input{
        Endpoint:    "https://example.ebarimt.mn",
        PosNo:       "YOUR_POS_NUMBER",
        MerchantTin: "YOUR_MERCHANT_TIN",
        // Optional parameters
        DB:       nil, // Your GORM DB instance if you want to store receipts
        MailHost: "",  // SMTP host for email notifications
        MailPort: 0,   // SMTP port for email notifications
    })

    // Create a receipt
    response, err := client.Create(models.CreateInputModel{
        // Add your receipt details here
    })
    if err != nil {
        // Handle error
    }
    // Use the response
    fmt.Printf("Receipt created: %+v\n", response)
}
```

## 📚 Documentation

For detailed documentation and examples, please visit our [documentation](https://github.com/techpartners-asia/ebarimt-pos3-go/wiki).

## 📁 Project Structure

```
ebarimt-pos3-go/
├── 📂 constants/     # Constant definitions
├── 📂 files/        # File handling utilities
├── 📂 pos3/         # Core POS 3.0 implementation
├── 📂 services/     # Service layer implementations
├── 📂 structs/      # Data structures
├── 📂 tests/        # Test files
└── 📂 utils/        # Utility functions
```

## ⚙️ Requirements

- 🔷 Go 1.23.5 or higher
- 🔷 GORM v1.25.12
- 🔷 Other dependencies as specified in go.mod

## 🤝 Contributing

We welcome contributions! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. 🍴 Fork the repository
2. 🌿 Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. 💾 Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. 📤 Push to the branch (`git push origin feature/AmazingFeature`)
5. 🔄 Open a Pull Request

## 🧪 Testing

Run the test suite:

```bash
go test ./...
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 💬 Support

For support, please open an issue in the GitHub repository or contact our support team.

## 🙏 Acknowledgments

- 👥 Thanks to all contributors who have helped shape this project
- 🌟 Special thanks to the Ebarimt team for their support and documentation

## 🔒 Security

For security concerns, please email security@techpartners.asia or open a security advisory in the GitHub repository.
