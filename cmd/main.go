package main

import (
	haftalik2 "gs/2haftalik"
	"gs/baza"
	"gs/booling"
	corss "gs/cors"
	dataspost "gs/datas_post"
	"gs/micro_gs_data_blok_read"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.Use(corss.CORSMiddleware())

	db, err := baza.ConnectionDb()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rdb := baza.ConnectDB()
	
	defer rdb.Close()

	newfunc := baza.NewBazaStruct(db, rdb)
	haftalik := haftalik2.NewHaftalik2Struct(db)

	go micro_gs_data_blok_read.StartFileWatcher()
	go dataspost.StartFileWatcher_datas()
	go haftalik.StartDatabaseListener()

	router.GET("/micro_gs_data_blok_read", micro_gs_data_blok_read.MicroGsDataBlokRead)
	router.POST("/micro_gs_data_blok_post", micro_gs_data_blok_read.MicroGsDataBlokPost)
	router.GET("/micro_gs_data_blok_ws", micro_gs_data_blok_read.WebSocketHandler)

	router.GET("/datas_get", dataspost.DatasRead)
	router.POST("/datas_post", dataspost.DatasPost)
	router.GET("/datas_ws", dataspost.WebSocketHandler_datas)

	router.POST("/register", newfunc.Register)
	router.POST("/confirm", newfunc.ConfirmationRegister)
	router.POST("/admin-approve", newfunc.AdminApprove)
	router.POST("/login", newfunc.Login)
	router.POST("/delete", newfunc.Delete)
	router.POST("/get-email", newfunc.GetEmail)

	router.POST("/haftalik2post", haftalik.Haftalik2)
	router.GET("/haftalik2ws", haftalik.WebSocketHandler)
	router.GET("/haftalik2get", haftalik.Get2Haftalik)

	router.POST("/booling_post", booling.BoolingPost)
	router.GET("/booling_get", booling.BoolingRead)

	router.Run(":9090")
}
