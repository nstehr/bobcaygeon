package raop

type DecryptingAlacDecoder struct {
	aesKey string
	aesIv  string
}

func NewDecryptingAlacDecoder(aesKey string, aesIv string) *DecryptingAlacDecoder {
	return &DecryptingAlacDecoder{aesKey: aesKey, aesIv: aesIv}
}

func (d *DecryptingAlacDecoder) Decode(data []byte) []byte {
	//audio := data[12:]
	return nil
}
