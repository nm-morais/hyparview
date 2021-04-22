package protocol

import "math/rand"

func getRandInt(roof int) int {
	return rand.Intn(roof)
}
