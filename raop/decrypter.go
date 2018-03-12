package raop

import (
	"crypto/aes"
	"crypto/cipher"
)

// AesDecrypter decoder capable of decoding the encrypted packet and treating it as ALAC encoded
type AesDecrypter struct {
	aesKey []byte
	aesIv  []byte
}

// NewAesDecrypter Returns a new decoder that will unencrypt and decode the packet as a Apple Lossless encoded packet
func NewAesDecrypter(aesKey []byte, aesIv []byte) *AesDecrypter {
	return &AesDecrypter{aesKey: aesKey, aesIv: aesIv}
}

// Decode decodes the supplied data using AES
func (d *AesDecrypter) Decode(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(d.aesKey)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, d.aesIv)
	audio := data[12:]
	todec := audio
	for len(todec) >= aes.BlockSize {
		mode.CryptBlocks(todec[:aes.BlockSize], todec[:aes.BlockSize])
		todec = todec[aes.BlockSize:]
	}

	send := make([]byte, len(audio))
	copy(send, audio)

	return send, nil
}
