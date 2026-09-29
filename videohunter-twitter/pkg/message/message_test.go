package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildReply(t *testing.T) {
	link := "https://www.myvideohunter.com/prod/url/abc"

	for _, language := range []string{"pt", "es", "en"} {
		reply := BuildReply(language, link)
		assert.Contains(t, reply, link)
		assert.NotEmpty(t, reply)
	}
}

func TestBuildReply_FallsBackToEnglish(t *testing.T) {
	link := "https://www.myvideohunter.com/prod/url/abc"
	assert.Contains(t, BuildReply("unknown", link), link)
}
