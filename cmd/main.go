package main

import (
	dataspost "gs/datas_post"
	"gs/micro_gs_data_blok_read"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/micro_gs_data_blok_read", micro_gs_data_blok_read.MicroGsDataBlokRead)
	router.POST("/micro_gs_data_blok_post", micro_gs_data_blok_read.MicroGsDataBlokPost)

	router.GET("/datas_get", dataspost.DatasRead)
	router.POST("/datas_post", dataspost.DatasPost)

	router.Run(":8080")
}
