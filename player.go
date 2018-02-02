package bobcaygeon

import (
	"log"
	"strings"

	"github.com/hajimehoshi/oto"
	"github.com/nstehr/bobcaygeon/rtsp"
)

type LocalPlayer struct{}

func NewLocalPlayer() *LocalPlayer {
	return &LocalPlayer{}
}
func (*LocalPlayer) Play(session *rtsp.Session) {
	go func() {
		p, err := oto.NewPlayer(44100, 2, 2, 10000)
		if err != nil {
			log.Println("error initializing player", err)
			return
		}
		for d := range session.OutputChan {
			rtpmap := session.Description.Attributes["rtpmap"]
			if strings.Contains(rtpmap, "AppleLossless") {
				decoded, err := CodecMap["AppleLossless"](d)
				if err != nil {
					log.Println("Problem decoding packet")
				}
				p.Write(decoded)
			} else {
				p.Write(d)
			}

		}
	}()

}
