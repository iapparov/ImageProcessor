package web

import (
	httpSwagger "github.com/swaggo/http-swagger"
	wbgin "github.com/wb-go/wbf/ginext"
	_ "imageProcessor/docs"
)

func RegisterRoutes(engine *wbgin.Engine, handler *ImageHandler) {
	api := engine.Group("/api")
	{
		api.POST("/upload", handler.UploadImage)
		api.GET("/image/:id", handler.GetImage)
		api.DELETE("/image/:id", handler.DeleteImage)
		api.GET("/swagger/*any", func(c *wbgin.Context) {
			httpSwagger.WrapHandler(c.Writer, c.Request)
		})
	}
}
