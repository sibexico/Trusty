//go:build !windows

package storage

func protectData(plain []byte) ([]byte, error) {
	copyOut := make([]byte, len(plain))
	copy(copyOut, plain)
	return copyOut, nil
}

func unprotectData(protected []byte) ([]byte, error) {
	copyOut := make([]byte, len(protected))
	copy(copyOut, protected)
	return copyOut, nil
}
