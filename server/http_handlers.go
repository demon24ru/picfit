package server

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/mholt/binding"

	"github.com/disintegration/imaging"

	api "gopkg.in/fukata/golang-stats-api-handler.v1"

	"github.com/thoas/picfit"
	"github.com/thoas/picfit/constants"
	"github.com/thoas/picfit/failure"
	"github.com/thoas/picfit/payload"
)

type handlers struct {
	processor *picfit.Processor
}

func (h handlers) stats(c *gin.Context) {
	c.JSON(http.StatusOK, api.GetStats())
}

func (h handlers) internalError(c *gin.Context) {
	panic(errors.WithStack(fmt.Errorf("KO")))
}

// healthcheck displays an ok response for healthcheck
func (h handlers) healthcheck(startedAt time.Time) func(c *gin.Context) {
	return func(c *gin.Context) {
		now := time.Now().UTC()

		uptime := now.Sub(startedAt)

		c.JSON(http.StatusOK, gin.H{
			"started_at": startedAt.String(),
			"uptime":     uptime.String(),
			"status":     "Ok",
			"version":    constants.Version,
			"revision":   constants.Revision,
			"build_time": constants.BuildTime,
			"compiler":   constants.Compiler,
			"ip_address": c.ClientIP(),
		})
	}
}

// display displays and image using resizing parameters
func (h handlers) display(c *gin.Context) error {
	file, err := h.processor.ProcessContext(c,
		picfit.WithAsync(true),
		picfit.WithLoad(true))
	if err != nil {
		return err
	}

	for k, v := range file.Headers {
		c.Header(k, v)
	}

	c.Header("Cache-Control", "must-revalidate")

	c.Data(http.StatusOK, file.ContentType(), file.Content())

	return nil
}

// upload uploads an image to the destination storage
func (h handlers) upload(c *gin.Context) error {
	multipartPayload := new(payload.Multipart)
	errs := binding.Bind(c.Request, multipartPayload)
	if errs != nil {
		return errs
	}

	file, err := h.processor.Upload(c, multipartPayload)

	if err != nil {
		return err
	}

	img, err := imaging.Decode(bytes.NewReader(file.Source))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": file.Filename(),
		//"path":     file.Path(),
		"url": file.URL(),
		"w":   img.Bounds().Max.X,
		"h":   img.Bounds().Max.Y,
	})

	return nil
}

// delete deletes a file from storages
func (h handlers) delete(c *gin.Context) error {
	var (
		err            error
		params, exists = c.Get("parameters")
		key, _         = c.Get("key")
	)
	if !exists {
		return failure.ErrUnprocessable
	}
	qs := params.(map[string]interface{})
	_, ok := qs[constants.OperationParamName].(string)
	if !ok {
		path, ok := qs["path"].(string)
		if !ok {
			return failure.ErrUnprocessable
		}
		err = h.processor.Delete(path)
	} else {
		err = h.processor.DeleteChild(key.(string))
	}

	if err != nil {
		return err
	}

	c.String(http.StatusOK, "Ok")

	return nil
}

// get generates an image synchronously and return its information from storages
func (h handlers) get(c *gin.Context) error {
	file, err := h.processor.ProcessContext(c,
		picfit.WithAsync(false),
		picfit.WithLoad(false))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": file.Filename(),
		//"path":     file.Path(),
		"url": file.URL(),
		"key": file.Key,
	})

	return nil
}

// redirect redirects to the image using base url from storage
func (h handlers) redirect(c *gin.Context) error {
	file, err := h.processor.ProcessContext(c,
		picfit.WithAsync(false),
		picfit.WithLoad(false))
	if err != nil {
		return err
	}

	c.Redirect(http.StatusMovedPermanently, file.URL())

	return nil
}

func pprofHandler(h http.HandlerFunc) gin.HandlerFunc {
	handler := http.HandlerFunc(h)
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
