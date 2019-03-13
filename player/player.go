package player

import (
	"bytes"
	"encoding/binary"
	"log"
	"sync"

	"github.com/hajimehoshi/oto"
	"github.com/nstehr/bobcaygeon/rtsp"
)

// Player defines a player for outputting the data packets from the session
type Player interface {
	Play(session *rtsp.Session)
	SetVolume(volume float64)
}

// LocalPlayer is a player that will just play the audio locally
type LocalPlayer struct {
	volLock sync.RWMutex
	volume  float64
}

// NewLocalPlayer instantiates a new LocalPlayer
func NewLocalPlayer() *LocalPlayer {
	return &LocalPlayer{volume: 1}
}

// Play will play the packets received on the specified session
func (lp *LocalPlayer) Play(session *rtsp.Session) {
	go lp.playStream(session)
}

// SetVolume accepts a float between 0 (mute) and 1 (full volume)
func (lp *LocalPlayer) SetVolume(volume float64) {
	lp.volLock.Lock()
	defer lp.volLock.Unlock()
	lp.volume = volume

}

func (lp *LocalPlayer) playStream(session *rtsp.Session) {
	p, err := oto.NewPlayer(44100, 2, 2, 10000)
	if err != nil {
		log.Println("error initializing player", err)
		return
	}
	decoder := GetCodec(session)
	for d := range session.DataChan {
		lp.volLock.RLock()
		vol := lp.volume
		lp.volLock.RUnlock()
		decoded, err := decoder(d)
		if err != nil {
			log.Println("Problem decoding packet")
		}
		p.Write(AdjustAudio(decoded, vol))
	}
}

// AdjustAudio takes a raw data frame of audio and a volume value between 0 and 1, 1 being full volume, 0 being mute
func AdjustAudio(raw []byte, vol float64) []byte {
	if vol == 1 {
		return raw
	}
	adjusted := new(bytes.Buffer)
	for i := 0; i < len(raw); i = i + 2 {
		var val int16
		b := raw[i : i+2]
		buf := bytes.NewReader(b)
		err := binary.Read(buf, binary.LittleEndian, &val)
		if err != nil {
			log.Println(err)
		}
		mod := vol * float64(val)
		val = int16(mod)
		val = min(32767, val)
		val = max(-32767, val)
		binary.Write(adjusted, binary.LittleEndian, val)

	}

	return adjusted.Bytes()
}

func min(a, b int16) int16 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int16) int16 {
	if a > b {
		return a
	}
	return b
}
