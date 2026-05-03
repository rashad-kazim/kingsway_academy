package files

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

const (
	profileImageSize    = 768
	profileJPEGQuality  = 85
	profileImageMIME    = "image/jpeg"
	profileImageExt     = ".jpg"
	profileImageDefault = "profile-photo.jpg"
)

type optimizedImage struct {
	content  []byte
	filename string
	mimeType string
}

func optimizeProfileImage(original []byte, filename string) (optimizedImage, error) {
	src, err := imaging.Decode(bytes.NewReader(original), imaging.AutoOrientation(true))
	if err != nil {
		return optimizedImage{}, err
	}

	dst := imaging.Fill(
		src,
		profileImageSize,
		profileImageSize,
		imaging.Center,
		imaging.Lanczos,
	)

	var out bytes.Buffer
	if err := imaging.Encode(&out, dst, imaging.JPEG, imaging.JPEGQuality(profileJPEGQuality)); err != nil {
		return optimizedImage{}, err
	}

	return optimizedImage{
		content:  out.Bytes(),
		filename: optimizedProfileFilename(filename),
		mimeType: profileImageMIME,
	}, nil
}

func optimizedProfileFilename(filename string) string {
	name := strings.TrimSpace(filename)
	if name == "" {
		return profileImageDefault
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if strings.TrimSpace(base) == "" {
		return profileImageDefault
	}

	return base + profileImageExt
}
