package engine

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/thoas/picfit/engine/backend"
	"github.com/thoas/picfit/engine/config"
	"github.com/thoas/picfit/image"
)

type Engine struct {
	DefaultFormat  string
	Format         string
	DefaultQuality int

	backends []Backend
}

type Backend struct {
	backend.Backend
	weight    int
	mimetypes []string
}

// New initializes an Engine
func New(cfg config.Config) *Engine {
	var b []Backend

	if cfg.Backends == nil {
		b = append(b, Backend{
			Backend:   &backend.GoImage{},
			mimetypes: MimeTypes,
		})
	} else {
		if cfg.Backends.Gifsicle != nil {
			path := cfg.Backends.Gifsicle.Path
			if path == "" {
				path = "gifsicle"
			}

			if _, err := exec.LookPath(path); err == nil {
				b = append(b, Backend{
					Backend:   &backend.Gifsicle{Path: path},
					mimetypes: cfg.Backends.Gifsicle.Mimetypes,
					weight:    cfg.Backends.Gifsicle.Weight,
				})
			}
		}
		if cfg.Backends.GoImage != nil {
			b = append(b, Backend{
				Backend:   &backend.GoImage{},
				mimetypes: cfg.Backends.GoImage.Mimetypes,
				weight:    cfg.Backends.GoImage.Weight,
			})
		}
		if cfg.Backends.Lilliput != nil {
			b = append(b, Backend{
				Backend:   backend.NewLilliput(cfg),
				mimetypes: cfg.Backends.Lilliput.Mimetypes,
				weight:    cfg.Backends.Lilliput.Weight,
			})
		}
	}

	sort.Slice(b, func(i, j int) bool {
		return b[i].weight < b[j].weight
	})

	quality := config.DefaultQuality
	if cfg.Quality != 0 {
		quality = cfg.Quality
	}

	return &Engine{
		DefaultFormat:  cfg.DefaultFormat,
		Format:         cfg.Format,
		DefaultQuality: quality,
		backends:       b,
	}
}

func (e Engine) String() string {
	backendNames := []string{}
	for _, backend := range e.backends {
		backendNames = append(backendNames, backend.String())
	}

	return strings.Join(backendNames, " ")
}

func (e Engine) getBackend(output *image.ImageFile) (*Backend, error) {
	var (
		err error
	)

	ct := output.ContentType()
	for j := range e.backends {
		for k := range e.backends[j].mimetypes {
			if ct == e.backends[j].mimetypes[k] {
				return &e.backends[j], nil
			}
		}
	}

	return nil, err
}

func (e Engine) UploadTransform(output *image.ImageFile, options *backend.Options) (*image.ImageFile, int, int, error) {
	var (
		err       error
		processed []byte
		source    = output.Source
	)

	bcnd, err := e.getBackend(output)
	if err != nil {
		return nil, 0, 0, err
	}

	processed, width, height, err := bcnd.UploadResize(output, options)
	if err != nil {
		return nil, 0, 0, err
	}

	output.Source = source
	output.Processed = processed

	return output, width, height, nil
}

func (e Engine) Transform(output *image.ImageFile, operations []EngineOperation) (*image.ImageFile, error) {
	var (
		err       error
		processed []byte
		source    = output.Source
	)

	for i := range operations {
		bcnd, err := e.getBackend(output)

		if err != nil {
			continue
		}

		processed, err = operate(bcnd, output, operations[i].Operation, operations[i].Options)
		if err == nil {
			output.Source = processed
			break
		}
		if err != backend.MethodNotImplementedError {
			return nil, err
		}
	}

	output.Source = source
	output.Processed = processed

	return output, err
}

func operate(b backend.Backend, img *image.ImageFile, operation Operation, options *backend.Options) ([]byte, error) {
	switch operation {
	case Flip:
		return b.Flip(img, options)
	case Rotate:
		return b.Rotate(img, options)
	case Resize:
		return b.Resize(img, options)
	case Thumbnail:
		return b.Thumbnail(img, options)
	case Fit:
		return b.Fit(img, options)
	case Flat:
		return b.Flat(img, options)
	case Blur:
		return b.Blur(img, options)
	default:
		return nil, fmt.Errorf("Operation not found for %s", operation)
	}
}
