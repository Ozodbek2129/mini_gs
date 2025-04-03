package main

import (
	minigs12 "gs/1_2_minigs"
	"gs/python_error"
	// haftalik2 "gs/2haftalik"
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
	// haftalik := haftalik2.NewHaftalik2Struct(db)

	go micro_gs_data_blok_read.StartFileWatcher()
	go dataspost.StartFileWatcher_datas()
	// go haftalik.StartDatabaseListener()
	go minigs12.StartFileWatcher_minigs12()
	go python_error.StartFileWatcher_Python()
	go booling.StartFileWatcher_Python_Bool()
	go booling.StartFileWatcher_Python_kamera()

	router.GET("/micro_gs_data_blok_read", micro_gs_data_blok_read.MicroGsDataBlokRead)
	router.POST("/micro_gs_data_blok_post", micro_gs_data_blok_read.MicroGsDataBlokPost)
	router.GET("/micro_gs_data_blok_ws", micro_gs_data_blok_read.WebSocketHandler)

	router.POST("/micro_gs_data_blok_post1",micro_gs_data_blok_read.MicroGsDataBlokPost1)
	router.GET("/micro_gs_data_blok_read1",micro_gs_data_blok_read.MicroGsDataBlokRead1)
	router.GET("/micro_gs_data_blok_ws1",micro_gs_data_blok_read.WebSocketHandler1)

	router.GET("/datas_get", dataspost.DatasRead)
	router.POST("/datas_post", dataspost.DatasPost)
	router.GET("/datas_ws", dataspost.WebSocketHandler_datas)

	router.POST("/register", newfunc.Register)
	router.POST("/confirm", newfunc.ConfirmationRegister)
	router.POST("/admin-approve", newfunc.AdminApprove)
	router.POST("/login", newfunc.Login)
	router.POST("/delete", newfunc.Delete)
	router.POST("/get-email", newfunc.GetEmail)

	// router.POST("/haftalik2post", haftalik.Haftalik2)
	// router.GET("/haftalik2ws", haftalik.WebSocketHandler)
	// router.GET("/haftalik2get", haftalik.Get2Haftalik)

	router.POST("/booling_post", booling.BoolingPost)
	router.GET("/booling_get", booling.BoolingRead)
	router.GET("/booling_ws", booling.WebSocketHandler_Python_kamera)

	router.POST("/minigs12_post",minigs12.Minigs12Post)
	router.GET("/minigs12_get",minigs12.Minigs12Read)
	router.GET("/minigs12_ws",minigs12.WebSocketHandler_minigs12)

	router.POST("/python_error_post", python_error.Python_error)
	router.GET("/python_error_get", python_error.Python_error_read)
	router.GET("/python_error_ws", python_error.WebSocketHandler_Python)

	router.POST("/python_bool",booling.BoolingPostPython)
	router.GET("/python_read",booling.BoolingReadPython)
	router.GET("/python_ws",booling.WebSocketHandler_Python_Bool)

	router.Run(":9090")
}
