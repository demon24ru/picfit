package backend

import (
	"bytes"
	"errors"
	"fmt"
	"image/gif"
	"math"
	"os/exec"

	"github.com/thoas/picfit/image"
)

// Gifsicle is the gifsicle backend.
type Gifsicle struct {
	Path string
}

func (b *Gifsicle) String() string {
	return "gifsicle"
}

// Fit implements Backend.
func (b *Gifsicle) Fit(*image.ImageFile, *Options) ([]byte, error) {
	return nil, MethodNotImplementedError
}

// Flat implements Backend.
func (b *Gifsicle) Flat(*image.ImageFile, *Options) ([]byte, error) {
	return nil, MethodNotImplementedError
}

// Flip implements Backend.
func (b *Gifsicle) Flip(*image.ImageFile, *Options) ([]byte, error) {
	return nil, MethodNotImplementedError
}

// Blur implements Backend.
func (b *Gifsicle) Blur(*image.ImageFile, *Options) ([]byte, error) {
	return nil, MethodNotImplementedError
}

// Resize implements Backend.
func (b *Gifsicle) Resize(imgfile *image.ImageFile, opts *Options) ([]byte, error) {
	cmd := exec.Command(b.Path,
		"--resize", fmt.Sprintf("%dx%d", opts.Width, opts.Height),
	)
	cmd.Stdin = bytes.NewReader(imgfile.Source)
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	var target *exec.ExitError
	if err := cmd.Run(); errors.As(err, &target) && target.Exited() {
		return nil, errors.New(stderr.String())
	} else if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

// UploadResize implements Backend.
func (b *Gifsicle) UploadResize(imgfile *image.ImageFile, opts *Options) ([]byte, int, int, error) {
	img, err := gif.Decode(bytes.NewReader(imgfile.Source))
	if err != nil {
		return nil, 0, 0, err
	}
	bounds := img.Bounds()
	if float64(bounds.Dx()) == math.Max(float64(bounds.Dx()), float64(bounds.Dy())) {
		opts.Width = 2000
	} else {
		opts.Height = 2000
	}
	cmd := exec.Command(b.Path,
		"--resize", fmt.Sprintf("%dx%d", opts.Width, opts.Height),
	)
	cmd.Stdin = bytes.NewReader(imgfile.Source)
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	var target *exec.ExitError
	if err := cmd.Run(); errors.As(err, &target) && target.Exited() {
		return nil, 0, 0, errors.New(stderr.String())
	} else if err != nil {
		return nil, 0, 0, err
	}
	img, err = gif.Decode(bytes.NewReader(stdout.Bytes()))
	if err != nil {
		return nil, 0, 0, err
	}
	return stdout.Bytes(), img.Bounds().Dx(), img.Bounds().Dy(), nil
}

// Rotate implements Backend.
func (b *Gifsicle) Rotate(*image.ImageFile, *Options) ([]byte, error) {
	return nil, MethodNotImplementedError
}

// Thumbnail implements Backend.
func (b *Gifsicle) Thumbnail(imgfile *image.ImageFile, opts *Options) ([]byte, error) {
	img, err := gif.Decode(bytes.NewReader(imgfile.Source))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	left, top, cropw, croph := computecrop(bounds.Dx(), bounds.Dy(), opts.Width, opts.Height)

	cmd := exec.Command(b.Path,
		"--crop", fmt.Sprintf("%d,%d+%dx%d", left, top, cropw, croph),
		"--resize", fmt.Sprintf("%dx%d", opts.Width, opts.Height),
	)
	cmd.Stdin = bytes.NewReader(imgfile.Source)
	stdout := new(bytes.Buffer)
	cmd.Stdout = stdout
	stderr := new(bytes.Buffer)
	cmd.Stderr = stderr

	var target *exec.ExitError
	if err := cmd.Run(); errors.As(err, &target) && target.Exited() {
		return nil, errors.New(stderr.String())
	} else if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func computecrop(srcw, srch, destw, desth int) (left, top, cropw, croph int) {
	srcratio := float64(srcw) / float64(srch)
	destratio := float64(destw) / float64(desth)

	if srcratio > destratio {
		cropw = int((destratio * float64(srch)) + 0.5)
		croph = srch
	} else {
		croph = int((float64(srcw) / destratio) + 0.5)
		cropw = srcw
	}

	left = int(float64(srcw-cropw) * 0.5)
	if left < 0 {
		left = 0
	}

	top = int(float64(srch-croph) * 0.5)
	if top < 0 {
		top = 0
	}
	return
}
