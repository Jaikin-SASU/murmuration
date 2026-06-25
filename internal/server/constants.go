package server

import "time"

// defaultTTL : un node est considéré mort si son heartbeat date de plus que ça.
// À ~3 s de cadence de heartbeat, 10 s tolère deux ratés réseau.
const defaultTTL = 10 * time.Second
