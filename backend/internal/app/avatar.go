package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ResizeAndEncodeAvatar decodes any supported image (JPEG, PNG, WebP),
// resizes it to maxDim resolution while preserving aspect ratio,
// encodes it as PNG, and computes its SHA256 hash.
func ResizeAndEncodeAvatar(r io.Reader, maxDim int) ([]byte, string, string, error) {
	srcImg, _, err := image.Decode(r)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := srcImg.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, "", "", fmt.Errorf("invalid image dimensions")
	}

	if maxDim <= 0 {
		maxDim = 1600
	}

	newW, newH := w, h
	if w > maxDim || h > maxDim {
		if w >= h {
			newW = maxDim
			newH = (h * maxDim) / w
		} else {
			newH = maxDim
			newW = (w * maxDim) / h
		}
	}

	dstImg := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dstImg, dstImg.Bounds(), srcImg, srcImg.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dstImg); err != nil {
		return nil, "", "", fmt.Errorf("failed to encode PNG avatar: %w", err)
	}

	data := buf.Bytes()
	hashBytes := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hashBytes[:])

	return data, "image/png", hashStr, nil
}
