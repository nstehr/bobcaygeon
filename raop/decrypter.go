package raop

import (
	"crypto/aes"
	"crypto/cipher"
)

// DecryptingDecoder decoder capable of decoding the encrypted packet and treating it as ALAC encoded
type DecryptingDecoder struct {
	aesKey []byte
	aesIv  []byte
}

// NewDecryptingAlacDecoder Returns a new decoder that will unencrypt and decode the packet as a Apple Lossless encoded packet
func NewDecryptingDecoder(aesKey []byte, aesIv []byte) *DecryptingDecoder {
	return &DecryptingDecoder{aesKey: aesKey, aesIv: aesIv}
}

func (d *DecryptingDecoder) Decode(data []byte) ([]byte, error) {
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
