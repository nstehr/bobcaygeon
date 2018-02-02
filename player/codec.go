package player

import "github.com/alicebob/alac"

type CodecHandler func(data []byte) ([]byte, error)

var CodecMap = map[string]CodecHandler{
	"AppleLossless": DecodeAlac}

func DecodeAlac(data []byte) ([]byte, error) {
	decoder, err := alac.New()
	if err != nil {
		return nil, err
	}
	return decoder.Decode(data), nil
}
