package player

import "github.com/alicebob/alac"

type codecHandler func(data []byte) ([]byte, error)

var codecMap = map[string]codecHandler{
	"AppleLossless": decodeAlac}

func decodeAlac(data []byte) ([]byte, error) {
	decoder, err := alac.New()
	if err != nil {
		return nil, err
	}
	return decoder.Decode(data), nil
}
