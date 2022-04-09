package api

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ClippedImage struct {
	TimeStamp   string
	ContentType string
	Data        []byte
}

func ParseClippedImage(data string) (image *ClippedImage, err error) {
	if strings.HasPrefix(data, "data:") {
		data = strings.TrimPrefix(data, "data:")
		split := strings.Split(data, ";")
		if len(split) == 2 && strings.HasPrefix(split[1], "base64,") {
			contentType := split[0]
			encodedImage := split[1][len("base64,"):]
			var imageBytes []byte
			if imageBytes, err = base64.StdEncoding.DecodeString(encodedImage); err != nil {
				return
			}
			ts := time.Now().Local().Format("20060102150405")
			ts += strings.ReplaceAll(uuid.NewString(), "-", "")
			ts = strings.ToUpper(ts)
			image = &ClippedImage{TimeStamp: ts, ContentType: contentType, Data: imageBytes}
		}
	}
	return
}
