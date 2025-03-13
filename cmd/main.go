package main

import (
	dataspost "gs/datas_post"
	"gs/micro_gs_data_blok_read"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	
	micro_gs_data_blok_read.StartFileWatcher()
	dataspost.StartFileWatcher_datas()
	router.GET("/micro_gs_data_blok_read", micro_gs_data_blok_read.MicroGsDataBlokRead)
	router.POST("/micro_gs_data_blok_post", micro_gs_data_blok_read.MicroGsDataBlokPost)
	router.GET("/micro_gs_data_blok_ws", micro_gs_data_blok_read.WebSocketHandler)

	router.GET("/datas_get", dataspost.DatasRead)
	router.POST("/datas_post", dataspost.DatasPost)
	router.GET("/datas_ws", dataspost.WebSocketHandler_datas)

	router.Run(":9090")
}
