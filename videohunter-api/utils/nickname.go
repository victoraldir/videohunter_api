package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// A nickname is the only name a chat room ever shows, so it is generated
// rather than derived from anything the account already has: publishing the
// email, or the local part of it, would put a real address in front of
// strangers on a public page.
//
// The two lists are deliberately ordinary words. The point is a label that is
// easy to tell apart from another one, not a handle that means something.
var nicknameAdjectives = []string{
	"Bold", "Brave", "Bright", "Calm", "Clever", "Cosmic", "Frosty", "Golden",
	"Happy", "Lucky", "Misty", "Noble", "Quiet", "Rapid", "Rusty", "Sharp",
	"Silver", "Sunny", "Swift", "Wild",
}

var nicknameAnimals = []string{
	"Badger", "Comet", "Falcon", "Fox", "Heron", "Lynx", "Moose", "Otter",
	"Owl", "Panda", "Puffin", "Raven", "Salmon", "Sparrow", "Tiger", "Turtle",
	"Walrus", "Wombat", "Yak", "Zebra",
}

// NewNickname returns a label like "SwiftOtter42". The number is random too,
// so two accounts that draw the same two words still read differently.
func NewNickname() string {
	adjective := pickWord(nicknameAdjectives)
	animal := pickWord(nicknameAnimals)

	number, err := rand.Int(rand.Reader, big.NewInt(90))
	if err != nil {
		// crypto/rand does not fail in practice, and the caller cannot do
		// anything useful with an error here, so a fixed suffix keeps the
		// nickname valid and unique enough rather than failing the request.
		return adjective + animal + "10"
	}

	return fmt.Sprintf("%s%s%d", adjective, animal, number.Int64()+10)
}

func pickWord(words []string) string {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
	if err != nil {
		return words[0]
	}

	return words[index.Int64()]
}
