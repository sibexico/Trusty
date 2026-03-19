![Go Version](https://img.shields.io/badge/Go-1.26.-blue?labelColor=gray&logo=go)
 [![Go Report Card](https://goreportcard.com/badge/github.com/sibexico/Trusty)](https://goreportcard.com/report/github.com/sibexico/Trusy)

# Trusty

Encrypt your conversations EVERYWHERE! Regardless of the communication method you use, it's possible to establish end-to-end encrypted conversations! You can use any messenger, send letters, use carrier pigeons - whatever you want. Just use the simple wizard for the key exchange process and start an encrypted conversation.

## Features
- Cryptographic functions
- Intuitive GUI for user interaction
- Modular code structure

## Missing Features
- Encrypted storage **WARNING!!! In the current version, messages are stored in plain text and can be read by anyone who has access to your PC! This needs to be addressed.**
- Support for different encryption algorithms

## Prerequisites
- [Go](https://golang.org/dl/) 1.26 or newer

## Getting Started
1. **Install with go install:**
   ```pwsh
   go install github.com/sibexico/Trusty@latest
   ```

2. **Or clone and run locally:**
   ```pwsh
   git clone https://github.com/sibexico/Trusty.git
   cd Trusty
   go run .
   ```

3. **Build manually:**
   ```pwsh
   go build -o trusty.exe
   .\trusty.exe
   ```

## Usage
Follow the on-screen instructions in the GUI to perform the key exchange and be able to use encrypted conversation.

## License
This project is licensed under the MIT License.
