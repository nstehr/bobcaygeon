package player

import (
	"log"
	"strings"

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
	go func() {
		p, err := oto.NewPlayer(44100, 2, 2, 10000)
		if err != nil {
			log.Println("error initializing player", err)
			return
		}
		var decoder codecHandler
		rtpmap := session.Description.Attributes["rtpmap"]
		if strings.Contains(rtpmap, "AppleLossless") {
			decoder = codecMap["AppleLossless"]
		} else {
			decoder = func(data []byte) ([]byte, error) { return data, nil }
		}
		for d := range session.OutputChan {
			decoded, err := decoder(d)
			if err != nil {
				log.Println("Problem decoding packet")
			}
			p.Write(decoded)
		}
	}()

}
