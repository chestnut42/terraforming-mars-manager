package docs

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

const (
	swaggerPath = "swagger.json"
	indexPath   = "index.html"
)

//go:embed index.tpl
var indexTemplate string

type Service struct {
	swaggerHandler http.Handler
	indexHandler   http.Handler
}

func NewService() (*Service, error) {
	tpl, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, map[string]interface{}{
		"swaggerUrl": swaggerPath,
	}); err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	return &Service{
		swaggerHandler: NewStaticHandler(api.SwaggerJson),
		indexHandler:   NewStaticHandler(buf.Bytes()),
	}, nil
}

func (s *Service) ConfigureRouter(mux *http.ServeMux, basePath string) {
	mux.Handle("GET "+filepath.Join("/", basePath, swaggerPath), s.swaggerHandler)
	mux.Handle("GET "+filepath.Join("/", basePath, indexPath), s.indexHandler)
}
