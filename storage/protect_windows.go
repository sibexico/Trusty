//go:build windows

package storage

import (
	"fmt"
	"syscall"
	"unsafe"
)

const cryptProtectUIForbidden = 0x1

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32              = syscall.NewLazyDLL("Crypt32.dll")
	kernel32             = syscall.NewLazyDLL("Kernel32.dll")
	procCryptProtectData = crypt32.NewProc("CryptProtectData")
	procCryptUnprotect   = crypt32.NewProc("CryptUnprotectData")
	procLocalFree        = kernel32.NewProc("LocalFree")
)

func bytesToBlob(data []byte) dataBlob {
	if len(data) == 0 {
		return dataBlob{}
	}
	return dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
}

func blobToBytes(blob dataBlob) []byte {
	if blob.cbData == 0 || blob.pbData == nil {
		return nil
	}
	return unsafe.Slice(blob.pbData, blob.cbData)
}

func freeBlob(blob dataBlob) {
	if blob.pbData != nil {
		procLocalFree.Call(uintptr(unsafe.Pointer(blob.pbData)))
	}
}

func protectData(plain []byte) ([]byte, error) {
	input := bytesToBlob(plain)
	var output dataBlob

	r1, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&input)),
		0,
		0,
		0,
		0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&output)),
	)
	if r1 == 0 {
		if err != nil {
			return nil, fmt.Errorf("protect data failed: %w", err)
		}
		return nil, fmt.Errorf("protect data failed")
	}
	defer freeBlob(output)

	protected := blobToBytes(output)
	copyOut := make([]byte, len(protected))
	copy(copyOut, protected)
	return copyOut, nil
}

func unprotectData(protected []byte) ([]byte, error) {
	input := bytesToBlob(protected)
	var output dataBlob

	r1, _, err := procCryptUnprotect.Call(
		uintptr(unsafe.Pointer(&input)),
		0,
		0,
		0,
		0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&output)),
	)
	if r1 == 0 {
		if err != nil {
			return nil, fmt.Errorf("unprotect data failed: %w", err)
		}
		return nil, fmt.Errorf("unprotect data failed")
	}
	defer freeBlob(output)

	plaintext := blobToBytes(output)
	copyOut := make([]byte, len(plaintext))
	copy(copyOut, plaintext)
	return copyOut, nil
}
