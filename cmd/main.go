package main

import (
	minigs12 "gs/1_2_minigs"
	fcmsignal "gs/FCM_signal"
	"gs/add_image"
	booling "gs/booling/bollling_kamera"
	"gs/malumotlar"
	microgsdatablokread1 "gs/micro_gs_data_blok_read_1"
	"gs/monitoring"
	"gs/python_error"
	"gs/serena"

	// haftalik2 "gs/2haftalik"
	"gs/baza"
	booling_kamera "gs/booling"
	corss "gs/cors"
	dataspost "gs/datas_post"
	"gs/micro_gs_data_blok_read"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"

	_ "gs/cmd/docs"
)

// @title        Google_docs_user API
// @version      1.0
// @description  This is an API for e-commerce platform.
// @termsOfService http://swagger.io/terms/
// @contact.name  API Support
// @contact.email support@swagger.io
// @BasePath      /
func main() {
	router := gin.Default()

	router.Use(corss.CORSMiddleware())
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	db, err := baza.ConnectionDb()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rdb := baza.ConnectDB()

	defer rdb.Close()

	newfunc := baza.NewBazaStruct(db, rdb)
	// haftalik := haftalik2.NewHaftalik2Struct(db)
	monitor := monitoring.NewBazaStructMonitor(db)

	fcm := fcmsignal.NewBazaFcmStruct(db)
	malumotlarr := malumotlar.NewMalumotlarRepo(db)

	go micro_gs_data_blok_read.StartWatcherMicroGs()
	go microgsdatablokread1.StartWatcherMicroGs1()
	go dataspost.StartFileWatcher_datas()
	// go haftalik.StartDatabaseListener()
	go minigs12.StartFileWatcher_minigs12()
	go python_error.StartFileWatcher_Python()
	go booling.StartFileWatcher_Python_Bool()
	go booling_kamera.StartFileWatcher_Python_kamera()
	go newfunc.WatchDatabase()
	go serena.StartFileWatcher_serena()

	go dataspost.StartTimeoutChecker()
	fcmsignal.InitFirebase()

	router.GET("/micro_gs_data_blok_read", micro_gs_data_blok_read.MicroGsDataBlokRead)
	router.POST("/micro_gs_data_blok_post", micro_gs_data_blok_read.MicroGsDataBlokPost)
	router.GET("/micro_gs_data_blok_ws", micro_gs_data_blok_read.WebSocketHandler)

	router.POST("/micro_gs_data_blok_post1", microgsdatablokread1.MicroGsDataBlokPost1)
	router.GET("/micro_gs_data_blok_read1", microgsdatablokread1.MicroGsDataBlokRead1)
	router.GET("/micro_gs_data_blok_ws1", microgsdatablokread1.WebSocketHandler1)

	router.GET("/datas_get", dataspost.DatasRead)
	router.POST("/datas_post", dataspost.DatasPost)
	router.GET("/datas_ws", dataspost.WebSocketHandler_datas)

	router.POST("/register", newfunc.Register)
	router.POST("/confirm", newfunc.ConfirmationRegister)
	router.POST("/admin-approve", newfunc.AdminApprove)
	router.GET("/login", newfunc.Login)
	router.DELETE("/delete", newfunc.Delete)
	router.POST("/get-email", newfunc.GetEmail)
	router.PUT("/active", newfunc.Active)
	router.GET("/getall", newfunc.GetAll)
	router.GET("/getall_ws", newfunc.HandleWebSocket)

	router.POST("/upload_image", add_image.UploadMedia)
	router.POST("/post_monitor_download", monitor.HandleExportMonitoring)
	router.POST("/post_monitor", monitor.CreateMonitoring)

	// router.POST("/haftalik2post", haftalik.Haftalik2)
	// router.GET("/haftalik2ws", haftalik.WebSocketHandler)
	// router.GET("/haftalik2get", haftalik.Get2Haftalik)

	router.POST("/booling_post", booling_kamera.BoolingPost)
	router.GET("/booling_get", booling_kamera.BoolingRead)
	router.GET("/booling_ws", booling_kamera.WebSocketHandler_Python_kamera)

	router.POST("/minigs12_post", minigs12.Minigs12Post)
	router.GET("/minigs12_get", minigs12.Minigs12Read)
	router.GET("/minigs12_ws", minigs12.WebSocketHandler_minigs12)

	router.POST("/python_error_post", python_error.Python_error)
	router.GET("/python_error_get", python_error.Python_error_read)
	router.GET("/python_error_ws", python_error.WebSocketHandler_Python)

	router.POST("/python_bool", booling.BoolingPostPython)
	router.GET("/python_read", booling.BoolingReadPython)
	router.GET("/python_ws", booling.WebSocketHandler_Python_Bool)

	router.POST("/serena_post", serena.SerenaPost)
	router.GET("/serena_get", serena.SerenaGet)
	router.GET("/serena_ws", serena.WebSocketHandler_serena)

	router.POST("/registerfcm", fcm.RegisterHandler)
	router.GET("/notify", fcm.NotifyAllHandler)

	router.POST("/malumotlarpost", malumotlarr.MalumotlarPost)
	router.GET("/malumotlarget", malumotlarr.MalumotlarGet)

	router.Run(":9090")
}
