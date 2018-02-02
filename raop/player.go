package raop

import (
	"github.com/nstehr/bobcaygeon/rtsp"
)

// Player defines a player for outputting the data packets from the session
type Player interface {
	Play(session *rtsp.Session)
}
