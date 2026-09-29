package message

import (
	"fmt"
	"math/rand"
)

// replies holds localized reply templates. The %s placeholder is the download
// link. Several variants per language keep back-to-back replies from being
// rejected by X as duplicate content (error 187 / HTTP 409).
var replies = map[string][]string{
	"pt": {
		"Olá 👋 Aqui está o seu vídeo:\n%s",
		"Prontinho! Baixe o seu vídeo aqui:\n%s",
		"O seu download está pronto 👇\n%s",
	},
	"es": {
		"Hola 👋 Aquí tienes tu vídeo:\n%s",
		"¡Listo! Descarga tu vídeo aquí:\n%s",
		"Tu descarga está lista 👇\n%s",
	},
	"en": {
		"Hi 👋 Here is your video:\n%s",
		"All set! Download your video here:\n%s",
		"Your download is ready 👇\n%s",
	},
}

// BuildReply returns a localized reply containing the download link.
func BuildReply(language, link string) string {
	templates := replies[language]
	if len(templates) == 0 {
		templates = replies["en"]
	}

	return fmt.Sprintf(templates[rand.Intn(len(templates))], link)
}
