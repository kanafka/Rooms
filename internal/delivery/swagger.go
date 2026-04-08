package delivery

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

func swaggerHandler() http.Handler {
	return httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	)
}
