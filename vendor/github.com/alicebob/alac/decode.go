/*
 * Translated from C to Go, this is the original license:
 *
 * ALAC (Apple Lossless Audio Codec) decoder
 * Copyright (c) 2005 David Hammerton
 * All rights reserved.
 *
 * This is the actual decoder.
 *
 * http://crazney.net/programs/itunes/alac.html
 *
 * Permission is hereby granted, free of charge, to any person
 * obtaining a copy of this software and associated documentation
 * files (the "Software"), to deal in the Software without
 * restriction, including without limitation the rights to use,
 * copy, modify, merge, publish, distribute, sublicense, and/or
 * sell copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be
 * included in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
 * OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
 * HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
 * WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
 * OTHER DEALINGS IN THE SOFTWARE.
 *
 */

package alac

import (
	"fmt"
)

type Alac struct {
	input_buffer                []byte
	input_buffer_index          int // we rewind the buffer sometimes
	input_buffer_bitaccumulator int // used so we can do arbitary bit reads

	samplesize     int
	numchannels    int
	bytespersample int

	/* buffers */
	predicterror_buffer_a []int32
	predicterror_buffer_b []int32

	outputsamples_buffer_a []int32
	outputsamples_buffer_b []int32

	uncompressed_bytes_buffer_a []int32
	uncompressed_bytes_buffer_b []int32

	/* stuff from setinfo */
	setinfo_max_samples_per_frame uint32 /* 0x1000 = 4096 */ // max samples per frame?
	setinfo_7a                    uint8  /* 0x00 */
	setinfo_sample_size           uint8  /* 0x10 */
	setinfo_rice_historymult      uint8  /* 0x28 */
	setinfo_rice_initialhistory   uint8  /* 0x0a */
	setinfo_rice_kmodifier        uint8  /* 0x0e */
	setinfo_7f                    uint8  /* 0x02 */
	setinfo_80                    uint16 /* 0x00ff */
	setinfo_82                    uint32 /* 0x000020e7 */ // max sample size??
	setinfo_86                    uint32 /* 0x00069fe4 */ // bit rate (avarge)??
	setinfo_8a_rate               uint32 /* 0x0000ac44 */
	/* end setinfo stuff */

}

const host_bigendian = false

/*
#define _Swap32(v) do { \
                   v = (((v) & 0x000000FF) << 0x18) | \
                       (((v) & 0x0000FF00) << 0x08) | \
                       (((v) & 0x00FF0000) >> 0x08) | \
                       (((v) & 0xFF000000) >> 0x18); } while(0)

*/

/*
func _Swap16(v int16) int16 {
	return (((v) & 0x00FF) << 0x08) | (((v) & 0xFF00) >> 0x08)
}
*/

// TODO: figure out how to translate the C bitfield to Go
// struct {signed int x:24;} se_struct_24;
// #define SignExtend24(val) (se_struct_24.x = val)
func signExtend24(v int32) int32 {
	return v
}

func (alac *Alac) allocateBuffers() {
	alac.predicterror_buffer_a = make([]int32, alac.setinfo_max_samples_per_frame*4)
	alac.predicterror_buffer_b = make([]int32, alac.setinfo_max_samples_per_frame*4)

	alac.outputsamples_buffer_a = make([]int32, alac.setinfo_max_samples_per_frame*4)
	alac.outputsamples_buffer_b = make([]int32, alac.setinfo_max_samples_per_frame*4)

	alac.uncompressed_bytes_buffer_a = make([]int32, alac.setinfo_max_samples_per_frame*4)
	alac.uncompressed_bytes_buffer_b = make([]int32, alac.setinfo_max_samples_per_frame*4)
}

/*

void alac_set_info(alac_file *alac, char *inputbuffer)
{
  char *ptr = inputbuffer;
  ptr += 4; / * size * /
  ptr += 4; / * frma * /
  ptr += 4; / * alac * /
  ptr += 4; / * size * /
  ptr += 4; / * alac * /

  ptr += 4; / * 0 ? * /

  alac->setinfo_max_samples_per_frame = *(uint32_t*)ptr; / * buffer size / 2 ? * /
  if (!host_bigendian)
      _Swap32(alac->setinfo_max_samples_per_frame);
  ptr += 4;
  alac->setinfo_7a = *(uint8_t*)ptr;
  ptr += 1;
  alac->setinfo_sample_size = *(uint8_t*)ptr;
  ptr += 1;
  alac->setinfo_rice_historymult = *(uint8_t*)ptr;
  ptr += 1;
  alac->setinfo_rice_initialhistory = *(uint8_t*)ptr;
  ptr += 1;
  alac->setinfo_rice_kmodifier = *(uint8_t*)ptr;
  ptr += 1;
  alac->setinfo_7f = *(uint8_t*)ptr;
  ptr += 1;
  alac->setinfo_80 = *(uint16_t*)ptr;
  if (!host_bigendian)
      _Swap16(alac->setinfo_80);
  ptr += 2;
  alac->setinfo_82 = *(uint32_t*)ptr;
  if (!host_bigendian)
      _Swap32(alac->setinfo_82);
  ptr += 4;
  alac->setinfo_86 = *(uint32_t*)ptr;
  if (!host_bigendian)
      _Swap32(alac->setinfo_86);
  ptr += 4;
  alac->setinfo_8a_rate = *(uint32_t*)ptr;
  if (!host_bigendian)
      _Swap32(alac->setinfo_8a_rate);

  allocate_buffers(alac);

}
*/

// supports reading 1 to 16 bits, in big endian format
func (alac *Alac) readbits_16(bits int) uint32 {
	var result uint32

	result = (uint32(alac.input_buffer[alac.input_buffer_index]) << 16)
	// bug in the original
	if len(alac.input_buffer)-alac.input_buffer_index > 1 {
		result |= (uint32(alac.input_buffer[alac.input_buffer_index+1]) << 8)
	}
	// bug in the original
	if len(alac.input_buffer)-alac.input_buffer_index > 2 {
		result |= uint32(alac.input_buffer[alac.input_buffer_index+2])
	}

	/* shift left by the number of bits we've already read,
	 * so that the top 'n' bits of the 24 bits we read will
	 * be the return bits */
	result = result << uint(alac.input_buffer_bitaccumulator)

	result = result & 0x00ffffff

	/* and then only want the top 'n' bits from that, where
	 * n is 'bits' */
	result = result >> uint(24-bits)

	new_accumulator := (alac.input_buffer_bitaccumulator + bits)

	/* increase the buffer pointer if we've read over n bytes. */
	alac.input_buffer_index += new_accumulator >> 3

	/* and the remainder goes back into the bit accumulator */
	alac.input_buffer_bitaccumulator = (new_accumulator & 7)

	return result
}

// supports reading 1 to 32 bits, in big endian format
func (alac *Alac) readbits(bits int) uint32 {
	var result int32 = 0

	if bits > 16 {
		bits -= 16
		result = int32(alac.readbits_16(16) << uint(bits))
	}

	result |= int32(alac.readbits_16(bits))

	return uint32(result)
}

/* reads a single bit */
func (alac *Alac) readbit() int {
	result := int(alac.input_buffer[alac.input_buffer_index])
	result = result << uint(alac.input_buffer_bitaccumulator)
	result = result >> 7 & 1

	new_accumulator := int(alac.input_buffer_bitaccumulator + 1)
	alac.input_buffer_index += new_accumulator / 8
	alac.input_buffer_bitaccumulator = (new_accumulator % 8)

	return result
}

func (alac *Alac) unreadbits(bits int) {
	new_accumulator := int(alac.input_buffer_bitaccumulator - bits)

	alac.input_buffer_index += new_accumulator >> 3

	alac.input_buffer_bitaccumulator = (new_accumulator & 7)
	if alac.input_buffer_bitaccumulator < 0 {
		alac.input_buffer_bitaccumulator *= -1
	}
}

func count_leading_zeros(input int) int {
	output := 0
	curbyte := 0

	curbyte = input >> 24
	if curbyte > 0 {
		goto found
	}
	output += 8

	curbyte = input >> 16
	if curbyte&0xff > 0 {
		goto found
	}
	output += 8

	curbyte = input >> 8
	if curbyte&0xff > 0 {
		goto found
	}
	output += 8

	curbyte = input
	if curbyte&0xff > 0 {
		goto found
	}
	output += 8

	return output

found:
	if (curbyte & 0xf0) == 0 {
		output += 4
	} else {
		curbyte >>= 4
	}

	if curbyte&0x8 > 0 {
		return output
	}
	if curbyte&0x4 > 0 {
		return output + 1
	}
	if curbyte&0x2 > 0 {
		return output + 2
	}
	if curbyte&0x1 > 0 {
		return output + 3
	}

	/* shouldn't get here: */
	return output + 4
}

const rice_threshold = 8 // maximum number of bits for a rice prefix.

func (alac *Alac) entropyDecodeValue(
	readSampleSize int,
	k int,
	rice_kmodifier_mask int,
) int32 {
	x := int32(0) // decoded value

	// read x, number of 1s before 0 represent the rice value.
	for x <= rice_threshold && alac.readbit() != 0 {
		x++
	}

	if x > rice_threshold {
		// read the number from the bit stream (raw value)
		value := int32(alac.readbits(readSampleSize))

		// mask value
		value &= int32((uint32(0xffffffff) >> uint(32-readSampleSize)))

		x = value
	} else {
		if k != 1 {
			extraBits := int(alac.readbits(k))

			// x = x * (2^k - 1)
			x *= int32((((1 << uint(k)) - 1) & rice_kmodifier_mask))

			if extraBits > 1 {
				x += int32(extraBits - 1)
			} else {
				alac.unreadbits(1)
			}
		}
	}

	return x
}

func (alac *Alac) entropyRiceDecode(
	outputBuffer []int32,
	outputSize int,
	readSampleSize int,
	rice_initialhistory int,
	rice_kmodifier int,
	rice_historymult int,
	rice_kmodifier_mask int,
) {
	var (
		history      int = rice_initialhistory
		signModifier int = 0
	)

	for outputCount := 0; outputCount < outputSize; outputCount++ {
		var (
			decodedValue int32
			finalValue   int32
			k            int32
		)

		k = int32(31 - rice_kmodifier - count_leading_zeros((history>>9)+3))

		if k < 0 {
			k += int32(rice_kmodifier)
		} else {
			k = int32(rice_kmodifier)
		}

		// note: don't use rice_kmodifier_mask here (set mask to 0xFFFFFFFF)
		decodedValue = int32(alac.entropyDecodeValue(readSampleSize, int(k), 0xFFFFFFFF))

		decodedValue += int32(signModifier)
		finalValue = (decodedValue + 1) / 2 // inc by 1 and shift out sign bit
		if decodedValue&1 != 0 {            // the sign is stored in the low bit
			finalValue *= -1
		}

		outputBuffer[outputCount] = finalValue

		signModifier = 0

		// update history
		history += (int(decodedValue) * rice_historymult) -
			((history * rice_historymult) >> 9)

		if decodedValue > 0xFFFF {
			history = 0xFFFF
		}

		// special case, for compressed blocks of 0
		if (history < 128) && (outputCount+1 < outputSize) {
			var blockSize int32

			signModifier = 1

			k = int32(count_leading_zeros(history)) + ((int32(history) + 16) / 64) - 24

			// note: blockSize is always 16bit
			blockSize = int32(alac.entropyDecodeValue(16, int(k), rice_kmodifier_mask))

			// got blockSize 0s
			if blockSize > 0 {
				// memset(&outputBuffer[outputCount+1], 0, blockSize*sizeof(*outputBuffer))
				for i := outputCount + 1; i < outputCount+1+int(blockSize*4); i++ {
					outputBuffer[i] = 0
				}
				outputCount += int(blockSize)
			}

			if blockSize > 0xFFFF {
				signModifier = 0
			}

			history = 0
		}
	}
}

func sign_extended32(val int32, bits int) int32 {
	return ((val << uint(32-bits)) >> uint(32-bits))
}

func sign_only(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

func predictorDecompressFirAdapt(
	error_buffer []int32,
	buffer_out []int32,
	output_size int,
	readsamplesize int,
	predictor_coef_table [32]int16,
	predictor_coef_num int,
	predictor_quantitization int,
) {
	/* first sample always copies */
	// *buffer_out = *error_buffer;
	buffer_out[0] = error_buffer[0]

	if predictor_coef_num == 0 {
		if output_size <= 1 {
			return
		}
		// memcpy(buffer_out+1, error_buffer+1, (output_size-1) * 4);
		copy(buffer_out[1:], error_buffer[1:1+((output_size-1)*4)])
		return
	}

	if predictor_coef_num == 0x1f { /* 11111 - max value of predictor_coef_num */
		/* second-best case scenario for fir decompression,
		 * error describes a small difference from the previous sample only
		 */
		if output_size <= 1 {
			return
		}
		for i := 0; i < output_size-1; i++ {
			prev_value := buffer_out[i]
			error_value := error_buffer[i+1]
			buffer_out[i+1] = int32(sign_extended32((prev_value + error_value),
				readsamplesize))
		}
		return
	}

	/* read warm-up samples */
	if predictor_coef_num > 0 {
		for i := 0; i < predictor_coef_num; i++ {
			val := buffer_out[i] + error_buffer[i+1]

			val = sign_extended32(val, readsamplesize)

			buffer_out[i+1] = val
		}
	}

	/* general case */
	if predictor_coef_num > 0 {
		for i := predictor_coef_num + 1; i < output_size; i++ {
			var (
				sum       int = 0
				outval    int
				error_val = error_buffer[i]
			)

			for j := 0; j < predictor_coef_num; j++ {
				sum += int((buffer_out[predictor_coef_num-j] - buffer_out[0]) *
					int32(predictor_coef_table[j]))
			}

			outval = (1 << uint(predictor_quantitization-1)) + sum
			outval = outval >> uint(predictor_quantitization)
			outval = outval + int(buffer_out[0]) + int(error_val)
			outval = int(sign_extended32(int32(outval), readsamplesize))

			buffer_out[predictor_coef_num+1] = int32(outval)

			if error_val > 0 {
				var predictor_num int = predictor_coef_num - 1

				for predictor_num >= 0 && error_val > 0 {
					var val int = int(buffer_out[0] -
						buffer_out[predictor_coef_num-predictor_num])
					var sign int = sign_only(val)

					predictor_coef_table[predictor_num] -= int16(sign)

					val *= sign /* absolute value */

					error_val -= int32((val >> uint(predictor_quantitization)) *
						(predictor_coef_num - predictor_num))

					predictor_num--
				}
			} else if error_val < 0 {
				predictor_num := predictor_coef_num - 1

				for predictor_num >= 0 && error_val < 0 {
					val := int(buffer_out[0] -
						buffer_out[predictor_coef_num-predictor_num])
					sign := -sign_only(int(val))

					predictor_coef_table[predictor_num] -= int16(sign)

					val *= sign /* neg value */

					error_val -= int32((val >> uint(predictor_quantitization)) *
						(predictor_coef_num - predictor_num))

					predictor_num--
				}
			}

			buffer_out = buffer_out[1:]
		}
	}
}

func deinterlace_16(
	buffer_a, buffer_b []int32,
	buffer_out []byte, // was an []int16
	numchannels, numsamples int,
	interlacing_shift uint8,
	interlacing_leftweight uint8,
) {
	if numsamples <= 0 {
		return
	}

	/* weighted interlacing */
	if interlacing_leftweight != 0 {
		for i := 0; i < numsamples; i++ {
			var (
				difference, midright int32
				left                 int16
				right                int16
			)

			midright = buffer_a[i]
			difference = buffer_b[i]

			right = int16(midright - ((difference * int32(interlacing_leftweight)) >> interlacing_shift))
			left = right + int16(difference)

			/* output is always little endian */
			/* TODO
			if host_bigendian {
				_Swap16(left)
				_Swap16(right)
			}
			*/

			// buffer_out[i*numchannels] = left
			// buffer_out[i*numchannels+1] = right
			buffer_out[2*i*numchannels] = byte(left)
			buffer_out[2*i*numchannels+1] = byte(left >> 8)
			buffer_out[2*i*numchannels+2] = byte(right)
			buffer_out[2*i*numchannels+3] = byte(right >> 8)
		}

		return
	}

	/* otherwise basic interlacing took place */
	for i := 0; i < numsamples; i++ {
		var left, right int16

		left = int16(buffer_a[i])
		right = int16(buffer_b[i])

		/* output is always little endian */
		/* TODO
		if host_bigendian {
			_Swap16(left)
			_Swap16(right)
		}
		*/

		// buffer_out[i*numchannels] = left
		// buffer_out[i*numchannels+1] = right
		buffer_out[2*i*numchannels] = byte(left)
		buffer_out[2*i*numchannels+1] = byte(left >> 8)
		buffer_out[2*i*numchannels+2] = byte(right)
		buffer_out[2*i*numchannels+3] = byte(right >> 8)
	}
}

// note: translation untested
func deinterlace_24(
	buffer_a, buffer_b []int32,
	uncompressed_bytes int,
	uncompressed_bytes_buffer_a, uncompressed_bytes_buffer_b []int32,
	buffer_out []byte, // was a *void
	numchannels, numsamples int,
	interlacing_shift, interlacing_leftweight uint8,
) {
	if numsamples <= 0 {
		return
	}

	/* weighted interlacing */
	if interlacing_leftweight > 0 {
		for i := 0; i < numsamples; i++ {
			var left, right int32

			midright := buffer_a[i]
			difference := buffer_b[i]

			right = midright - ((difference * int32(interlacing_leftweight)) >> interlacing_shift)
			left = right + difference

			if uncompressed_bytes > 0 {
				mask := uint32(^(0xFFFFFFFF << uint(uncompressed_bytes*8)))
				left <<= uint(uncompressed_bytes * 8)
				right <<= uint(uncompressed_bytes * 8)

				left |= uncompressed_bytes_buffer_a[i] & int32(mask)
				right |= uncompressed_bytes_buffer_b[i] & int32(mask)
			}

			buffer_out[i*numchannels*3] = byte((left) & 0xFF)
			buffer_out[i*numchannels*3+1] = byte((left >> 8) & 0xFF)
			buffer_out[i*numchannels*3+2] = byte((left >> 16) & 0xFF)

			buffer_out[i*numchannels*3+3] = byte((right) & 0xFF)
			buffer_out[i*numchannels*3+4] = byte((right >> 8) & 0xFF)
			buffer_out[i*numchannels*3+5] = byte((right >> 16) & 0xFF)
		}

		return
	}

	/* otherwise basic interlacing took place */
	for i := 0; i < numsamples; i++ {
		left := buffer_a[i]
		right := buffer_b[i]

		if uncompressed_bytes > 0 {
			mask := uint32(^(0xFFFFFFFF << uint(uncompressed_bytes*8)))
			left <<= uint(uncompressed_bytes * 8)
			right <<= uint(uncompressed_bytes * 8)

			left |= uncompressed_bytes_buffer_a[i] & int32(mask)
			right |= uncompressed_bytes_buffer_b[i] & int32(mask)
		}

		buffer_out[i*numchannels*3] = byte((left) & 0xFF)
		buffer_out[i*numchannels*3+1] = byte((left >> 8) & 0xFF)
		buffer_out[i*numchannels*3+2] = byte((left >> 16) & 0xFF)

		buffer_out[i*numchannels*3+3] = byte((right) & 0xFF)
		buffer_out[i*numchannels*3+4] = byte((right >> 8) & 0xFF)
		buffer_out[i*numchannels*3+5] = byte((right >> 16) & 0xFF)

	}

}

func (alac *Alac) decodeFrame(inbuffer []byte) []byte {
	outputsamples := alac.setinfo_max_samples_per_frame

	/* setup the stream */
	alac.input_buffer = inbuffer
	alac.input_buffer_index = 0
	alac.input_buffer_bitaccumulator = 0

	channels := alac.readbits(3)

	outputsize := int(outputsamples) * alac.bytespersample

	switch channels {
	case 0: /* 1 channel */
		// note: translation untested
		var (
			readsamplesize int
			ricemodifier   int
		)

		// 2^result = something to do with output waiting.
		// perhaps matters if we read > 1 frame in a pass?
		alac.readbits(4)
		alac.readbits(12) // unknown, skip 12 bits

		var (
			hassize            = int(alac.readbits(1)) // the output sample size is stored soon
			uncompressed_bytes = int(alac.readbits(2)) // number of bytes in the (compressed) stream that are not compressed
			isnotcompressed    = int(alac.readbits(1)) // whether the frame is compressed
		)

		if hassize > 0 {
			// now read the number of samples, as a 32bit integer
			outputsamples = alac.readbits(32)
			outputsize = int(outputsamples) * alac.bytespersample
		}

		readsamplesize = int(alac.setinfo_sample_size) - (uncompressed_bytes * 8)

		if isnotcompressed == 0 {
			// so it is compressed
			var (
				predictor_coef_table [32]int16
			)

			// skip 16 bits, not sure what they are. seem to be used in
			// two channel case
			alac.readbits(8)
			alac.readbits(8)

			prediction_type := int(alac.readbits(4))
			prediction_quantitization := int(alac.readbits(4))
			ricemodifier = int(alac.readbits(3))
			predictor_coef_num := int(alac.readbits(5))
			// read the predictor table
			for i := 0; i < predictor_coef_num; i++ {
				predictor_coef_table[i] = int16(alac.readbits(16))
			}

			if uncompressed_bytes != 0 {
				for i := uint32(0); i < outputsamples; i++ {
					alac.uncompressed_bytes_buffer_a[i] = int32(alac.readbits(uncompressed_bytes * 8))
				}
			}

			alac.entropyRiceDecode(
				alac.predicterror_buffer_a,
				int(outputsamples),
				readsamplesize,
				int(alac.setinfo_rice_initialhistory),
				int(alac.setinfo_rice_kmodifier),
				ricemodifier*int(alac.setinfo_rice_historymult)/4,
				(1<<alac.setinfo_rice_kmodifier)-1,
			)

			if prediction_type == 0 {
				// adaptive fir
				predictorDecompressFirAdapt(
					alac.predicterror_buffer_a,
					alac.outputsamples_buffer_a,
					int(outputsamples),
					readsamplesize,
					predictor_coef_table,
					predictor_coef_num,
					prediction_quantitization,
				)
			} else {
				fmt.Printf("FIXME: unhandled predicition type: %d\n", prediction_type)
				// i think the only other prediction type (or perhaps this is just a
				// boolean?) runs adaptive fir twice.. like:
				// predictor_decompress_fir_adapt(predictor_error, tempout, ...)
				// predictor_decompress_fir_adapt(predictor_error, outputsamples ...)
				// little strange..
			}

		} else {
			// not compressed, easy case
			if alac.setinfo_sample_size <= 16 {
				for i := uint32(0); i < outputsamples; i++ {
					audiobits := int32(alac.readbits(int(alac.setinfo_sample_size)))
					audiobits = sign_extended32(audiobits, int(alac.setinfo_sample_size))

					alac.outputsamples_buffer_a[i] = audiobits
				}
			} else {
				for i := uint32(0); i < outputsamples; i++ {
					audiobits := int32(alac.readbits(16))
					// special case of sign extension..
					// as we'll be ORing the low 16bits into this
					audiobits = audiobits << (alac.setinfo_sample_size - 16)
					audiobits |= int32(alac.readbits(int(alac.setinfo_sample_size - 16)))
					audiobits = signExtend24(audiobits)

					alac.outputsamples_buffer_a[i] = audiobits
				}
			}
			uncompressed_bytes = 0 // always 0 for uncompressed
		}

		outbuffer := make([]byte, outputsize)
		switch alac.setinfo_sample_size {
		case 16:
			for i := uint32(0); i < outputsamples; i++ {
				sample := int16(alac.outputsamples_buffer_a[i])
				// TODO
				// if host_bigendian {
				// _Swap16(sample);
				// }

				// ((int16_t*)outbuffer)[i * alac->numchannels] = sample;
				outbuffer[2*int(i)*alac.numchannels] = byte(sample)
				outbuffer[2*int(i)*alac.numchannels+1] = byte(sample >> 8)
			}
		case 24:
			for i := uint32(0); i < outputsamples; i++ {
				sample := int32(alac.outputsamples_buffer_a[i])
				if uncompressed_bytes != 0 {
					sample = sample << uint(uncompressed_bytes*8)
					mask := uint32(^(0xFFFFFFFF << uint(uncompressed_bytes*8)))
					sample |= alac.uncompressed_bytes_buffer_a[i] & int32(mask)
				}

				outbuffer[int(i)*alac.numchannels*3] = byte((sample) & 0xFF)
				outbuffer[int(i)*alac.numchannels*3+1] = byte((sample >> 8) & 0xFF)
				outbuffer[int(i)*alac.numchannels*3+2] = byte((sample >> 16) & 0xFF)
			}
		case 20, 32:
			fmt.Printf("FIXME: unimplemented sample size %d\n", alac.setinfo_sample_size)
		default:
		}
		return outbuffer
	case 1:
		// 2 channels
		var (
			hassize         int
			isnotcompressed int
			readsamplesize  int

			uncompressed_bytes int

			interlacing_shift      uint8
			interlacing_leftweight uint8
		)

		/* 2^result = something to do with output waiting.
		 * perhaps matters if we read > 1 frame in a pass?
		 */
		alac.readbits(4)

		alac.readbits(12) /* unknown, skip 12 bits */

		hassize = int(alac.readbits(1)) /* the output sample size is stored soon */

		uncompressed_bytes = int(alac.readbits(2)) /* the number of bytes in the (compressed) stream that are not compressed */

		isnotcompressed = int(alac.readbits(1)) /* whether the frame is compressed */

		if hassize != 0 {
			/* now read the number of samples,
			 * as a 32bit integer */
			outputsamples = alac.readbits(32)
			outputsize = int(outputsamples) * alac.bytespersample
		}

		readsamplesize = int(alac.setinfo_sample_size) - (uncompressed_bytes * 8) + 1

		if isnotcompressed == 0 {
			/* compressed */
			interlacing_shift = uint8(alac.readbits(8))
			interlacing_leftweight = uint8(alac.readbits(8))
			var (
				predictor_coef_table_a [32]int16
				predictor_coef_table_b [32]int16
			)

			/******** channel 1 ***********/
			var (
				prediction_type_a           int = int(alac.readbits(4))
				prediction_quantitization_a int = int(alac.readbits(4))

				ricemodifier_a       int = int(alac.readbits(3))
				predictor_coef_num_a int = int(alac.readbits(5))
			)

			/* read the predictor table */
			for i := 0; i < predictor_coef_num_a; i++ {
				predictor_coef_table_a[i] = int16(alac.readbits(16))
			}

			/******** channel 2 *********/
			var (
				prediction_type_b           int = int(alac.readbits(4))
				prediction_quantitization_b int = int(alac.readbits(4))

				ricemodifier_b       int = int(alac.readbits(3))
				predictor_coef_num_b int = int(alac.readbits(5))
			)
			/* read the predictor table */
			for i := 0; i < predictor_coef_num_b; i++ {
				predictor_coef_table_b[i] = int16(alac.readbits(16))
			}

			/*********************/
			if uncompressed_bytes != 0 {
				/* see mono case */
				for i := uint32(0); i < outputsamples; i++ {
					alac.uncompressed_bytes_buffer_a[i] = int32(alac.readbits(uncompressed_bytes * 8))
					alac.uncompressed_bytes_buffer_b[i] = int32(alac.readbits(uncompressed_bytes * 8))
				}
			}

			/* channel 1 */
			alac.entropyRiceDecode(
				alac.predicterror_buffer_a,
				int(outputsamples),
				readsamplesize,
				int(alac.setinfo_rice_initialhistory),
				int(alac.setinfo_rice_kmodifier),
				ricemodifier_a*int(alac.setinfo_rice_historymult)/4,
				(1<<alac.setinfo_rice_kmodifier)-1)

			if prediction_type_a == 0 { /* adaptive fir */
				predictorDecompressFirAdapt(
					alac.predicterror_buffer_a,
					alac.outputsamples_buffer_a,
					int(outputsamples),
					readsamplesize,
					predictor_coef_table_a,
					predictor_coef_num_a,
					prediction_quantitization_a)
			} else {
				/* see mono case */
				fmt.Printf("FIXME: unhandled predicition type: %d\n", prediction_type_a)
			}
			/* channel 2 */
			alac.entropyRiceDecode(
				alac.predicterror_buffer_b,
				int(outputsamples),
				readsamplesize,
				int(alac.setinfo_rice_initialhistory),
				int(alac.setinfo_rice_kmodifier),
				ricemodifier_b*int(alac.setinfo_rice_historymult)/4,
				(1<<alac.setinfo_rice_kmodifier)-1)

			if prediction_type_b == 0 { /* adaptive fir */
				predictorDecompressFirAdapt(
					alac.predicterror_buffer_b,
					alac.outputsamples_buffer_b,
					int(outputsamples),
					readsamplesize,
					predictor_coef_table_b,
					predictor_coef_num_b,
					prediction_quantitization_b)
			} else {
				fmt.Printf("FIXME: unhandled predicition type: %d\n", prediction_type_b)
			}
		} else {
			/* not compressed, easy case */
			// note: translation untested
			if alac.setinfo_sample_size <= 16 {
				for i := uint32(0); i < outputsamples; i++ {
					audiobits_a := alac.readbits(int(alac.setinfo_sample_size))
					audiobits_b := alac.readbits(int(alac.setinfo_sample_size))

					audiobits_a = uint32(sign_extended32(int32(audiobits_a), int(alac.setinfo_sample_size)))
					audiobits_b = uint32(sign_extended32(int32(audiobits_b), int(alac.setinfo_sample_size)))

					alac.outputsamples_buffer_a[i] = int32(audiobits_a)
					alac.outputsamples_buffer_b[i] = int32(audiobits_b)
				}
			} else {
				for i := uint32(0); i < outputsamples; i++ {
					audiobits_a := int32(alac.readbits(16))
					audiobits_a = audiobits_a << (alac.setinfo_sample_size - 16)
					audiobits_a |= int32(alac.readbits(int(alac.setinfo_sample_size - 16)))
					audiobits_a = signExtend24(audiobits_a)

					audiobits_b := int32(alac.readbits(16))
					audiobits_b = audiobits_b << (alac.setinfo_sample_size - 16)
					audiobits_b |= int32(alac.readbits(int(alac.setinfo_sample_size - 16)))
					audiobits_b = signExtend24(audiobits_b)

					alac.outputsamples_buffer_a[i] = audiobits_a
					alac.outputsamples_buffer_b[i] = audiobits_b
				}
			}
			uncompressed_bytes = 0 // always 0 for uncompressed
			interlacing_shift = 0
			interlacing_leftweight = 0
		}

		outbuffer := make([]byte, outputsize)

		switch alac.setinfo_sample_size {
		case 16:
			deinterlace_16(
				alac.outputsamples_buffer_a,
				alac.outputsamples_buffer_b,
				outbuffer, // was []int16
				alac.numchannels,
				int(outputsamples),
				interlacing_shift,
				interlacing_leftweight,
			)
		case 24:
			deinterlace_24(
				alac.outputsamples_buffer_a,
				alac.outputsamples_buffer_b,
				uncompressed_bytes,
				alac.uncompressed_bytes_buffer_a,
				alac.uncompressed_bytes_buffer_b,
				outbuffer, // was []int16
				alac.numchannels,
				int(outputsamples),
				interlacing_shift,
				interlacing_leftweight,
			)
		case 20, 32:
			fmt.Printf("FIXME: unimplemented sample size %d\n", alac.setinfo_sample_size)
		default:
		}
		return outbuffer
	default:
		fmt.Printf("unimplemented channel size %d\n", channels+1)
	}

	return nil
}

func create_alac(samplesize, numchannels int) *Alac {
	return &Alac{
		samplesize:     samplesize,
		numchannels:    numchannels,
		bytespersample: (samplesize / 8) * numchannels,
	}
}
