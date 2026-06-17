package main

import (
	"beacon-system/config"
	"beacon-system/database"
	"beacon-system/handlers"
	"beacon-system/modules/alarm_mqtt"
	"beacon-system/modules/dtu_receiver"
	"beacon-system/modules/eventbus"
	"beacon-system/modules/network_reliability_analyzer"
	"beacon-system/modules/visibility_analyzer"
	"beacon-system/mqtt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	params := cfg.Params
	if params == nil {
		log.Println("Warning: params not loaded from config, using defaults")
	}

	if err := database.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()
	defer eventbus.Get().Close()

	var mqttClient *mqtt.Client
	var err error
	if cfg.DemoMode {
		log.Println("Running in demo mode, MQTT client disabled")
	} else {
		mqttClient, err = mqtt.New(cfg)
		if err != nil {
			log.Printf("Warning: Failed to connect to MQTT: %v", err)
			log.Println("Continuing without MQTT (alerts will only be stored in DB)")
		} else {
			defer mqttClient.Disconnect()
		}
	}

	dtuReceiver := dtu_receiver.New(cfg)
	visibilityAnalyzer := visibility_analyzer.New(cfg)
	networkAnalyzer := network_reliability_analyzer.New(cfg)
	alarmModule := alarm_mqtt.New(cfg, mqttClient)

	dtuReceiver.Start()
	visibilityAnalyzer.Start()
	networkAnalyzer.Start()
	alarmModule.Start()
	defer alarmModule.Stop()

	handlers.InitDTUReceiver(dtuReceiver)
	handlers.InitVisibilityAnalyzer(visibilityAnalyzer)
	handlers.InitNetworkModules(networkAnalyzer, alarmModule)
	handlers.InitHandlers()

	log.Println("[Init] All modules initialized: dtu_receiver, visibility_analyzer, network_reliability_analyzer, alarm_mqtt")
	if params != nil {
		log.Printf("[Init] Params loaded: DEM radius=%dm, ITU-R k=%.3f, MC iterations=%d, IS edge threshold=%d",
			int(params.Terrain.DEMSearchRadiusMeters),
			params.Atmosphere.EffectiveEarthFactorK,
			params.Reliability.DefaultMCIterations,
			params.Reliability.ISEdgeThreshold)
	} else {
		log.Println("[Init] Params not available (using defaults embedded in modules)")
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/beacons", handlers.GetBeacons)
		api.GET("/beacons/:id", handlers.GetBeacon)
		api.POST("/beacons", handlers.CreateBeacon)

		api.GET("/sensor-data", handlers.GetSensorData)
		api.POST("/sensor-data", handlers.PostSensorData)
		api.GET("/sensor-data/latest", handlers.GetLatestSensorData)

		api.GET("/signal-reception", handlers.GetSignalReception)
		api.POST("/signal-reception", handlers.PostSignalReception)

		api.GET("/visibility", handlers.GetVisibilityMatrix)
		api.GET("/visibility/calculate", handlers.CalculateVisibility)
		api.POST("/visibility/matrix", handlers.CalculateVisibilityMatrix)
		api.GET("/beacons/:id/viewshed", handlers.GetViewShed)

		api.GET("/network/topology", handlers.GetNetworkTopology)
		api.GET("/network/reliability", handlers.AnalyzeReliability)
		api.GET("/network/reliability/history", handlers.GetReliabilityHistory)
		api.GET("/network/connectivity", handlers.CheckConnectivity)
		api.GET("/network/critical-links", handlers.GetCriticalLinks)

		api.GET("/alerts", handlers.GetAlerts)
		api.PUT("/alerts/:id/resolve", handlers.ResolveAlert)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "beacon-visibility-analysis-system",
			"modules": []string{"dtu_receiver", "visibility_analyzer", "network_reliability_analyzer", "alarm_mqtt"},
		})
	})

	r.GET("/params", func(c *gin.Context) {
		c.JSON(200, params)
	})

	log.Printf("Server starting on port %s...", cfg.ServerPort)
	log.Printf("Demo mode: %v", cfg.DemoMode)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
