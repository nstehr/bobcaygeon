// Apple Lossless (ALAC) decoder
package alac

import (
	"fmt"
	"strconv"
	"strings"
)

// New alac decoder. Sample size 16, 2 chan!
func New() (*Alac, error) {
	a := create_alac(16, 2)
	if a == nil {
		return nil, fmt.Errorf("can't create alac. No idea why, though")
	}
	// TODO: fmtp stuff
	// fmtp: 96 352 0 16 40 10 14 2 255 0 0 44100
	a.setinfo_max_samples_per_frame = 352 // frame_size;
	a.setinfo_7a = 0                      // fmtp[2];
	a.setinfo_sample_size = 16            // sample_size;
	a.setinfo_rice_historymult = 40       // fmtp[4];
	a.setinfo_rice_initialhistory = 10    // fmtp[5];
	a.setinfo_rice_kmodifier = 14         // fmtp[6];
	a.setinfo_7f = 2                      // fmtp[7];
	a.setinfo_80 = 255                    // fmtp[8];
	a.setinfo_82 = 0                      // fmtp[9];
	a.setinfo_86 = 0                      // fmtp[10];
	a.setinfo_8a_rate = 44100             // fmtp[11];

	a.allocateBuffers()
	return a, nil
}

// New alac decoder. Sample size 16, 2 chan!
func NewFromFmtp(fmtp string) (*Alac, error) {
	a := create_alac(16, 2)
	if a == nil {
		return nil, fmt.Errorf("can't create alac. No idea why, though")
	}

	iv, err := strings2ints(strings.Split(fmtp, " "))
	if err != nil {
		return nil, err
	}

	// fmtp: 96 352 0 16 40 10 14 2 255 0 0 44100
	a.setinfo_max_samples_per_frame = uint32(iv[1]) // 24: frameLength uint32;
	a.setinfo_7a = uint8(iv[2])                     // 28: compatibleVersion byte;
	a.setinfo_sample_size = uint8(iv[3])            // 29: bitDepth byte;
	a.setinfo_rice_historymult = uint8(iv[4])       // 30: pb byte;
	a.setinfo_rice_initialhistory = uint8(iv[5])    // 31: mb byte;
	a.setinfo_rice_kmodifier = uint8(iv[6])         // 32: kb byte;
	a.setinfo_7f = uint8(iv[7])                     // 33:  numChannels byte;
	a.setinfo_80 = uint16(iv[8])                    // 34: maxRun uint16;
	a.setinfo_82 = uint32(iv[9])                    // 36: maxFrameBytes uint32;
	a.setinfo_86 = uint32(iv[10])                   // 40: avgBitRate uint32;
	a.setinfo_8a_rate = uint32(iv[11])              // 44: sampleRate uint32;

	a.allocateBuffers()
	return a, nil
}

/*
Decode a frame.
*/
func (a *Alac) Decode(f []byte) []byte {
	return a.decodeFrame(f)
}

/*
Return the number of bits per sample and channel.
*/
func (a *Alac) BitDepth() int {
	return int(a.samplesize)
}

/*
Return the number of channels to decode.
*/
func (a *Alac) NumChannels() int {
	return int(a.setinfo_7f)
}

/*
Return the SampleRate in samples per second.
*/
func (a *Alac) SampleRate() int {
	return int(a.setinfo_8a_rate)
}

// Convert an array of strings (FMTP) to ints.
func strings2ints(sv []string) (iv []int64, err error) {

	iv = make([]int64, len(sv))
	for ii, s := range sv {
		iv[ii], err = strconv.ParseInt(s, 10, 32)
		if err != nil {
			return
		}
	}
	return

}
