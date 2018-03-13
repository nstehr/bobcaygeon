package player

import (
	"log"

	"github.com/hajimehoshi/oto"
	"github.com/nstehr/bobcaygeon/rtsp"
)

// Player defines a player for outputting the data packets from the session
type Player interface {
	Play(session *rtsp.Session)
}

// LocalPlayer is a player that will just play the audio locally
type LocalPlayer struct{}

// NewLocalPlayer instantiates a new LocalPlayer
func NewLocalPlayer() *LocalPlayer {
	return &LocalPlayer{}
}

// Play will play the packets received on the specified session
func (*LocalPlayer) Play(session *rtsp.Session) {
	go playStream(session)
}

func playStream(session *rtsp.Session) {
	p, err := oto.NewPlayer(44100, 2, 2, 10000)
	if err != nil {
		log.Println("error initializing player", err)
		return
	}
	decoder := GetCodec(session)
	for d := range session.DataChan {
		decoded, err := decoder(d)
		if err != nil {
			log.Println("Problem decoding packet")
		}
		p.Write(decoded)
	}
}
