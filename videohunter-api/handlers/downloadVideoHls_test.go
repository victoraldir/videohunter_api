package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandle(t *testing.T) {
	t.Run("Should unmarshal request body", func(t *testing.T) {

		// Arrage
		data := `{ \"url\": \"https://v.redd.it/b4cikpfnw80d1/HLSPlaylist.m3u8\\?a\\=1719161822%2CYWZkNDY2Mjg2NGUxNGMyMDRiOTExZGEzYWFkYjJjNTE1MDQxYjVjMTY5NDE2MjU4OThjNjU4ZTM4MDhjM2JlMQ%3D%3D\\&amp\\;v\\=1\\&amp\\;f\\=sd\"}`
		downloadRequest := &DownalodRequest{}

		//Act

		// Unscape the string
		data = strings.Replace(data, "\\\"", "\"", -1)

		err := json.Unmarshal([]byte(data), downloadRequest)

		// Assert
		assert.Nil(t, err)

	})
}
