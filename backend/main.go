package main

import (
	"beacon-system/config"
	"beacon-system/database"
	"beacon-system/handlers"
	"beacon-system/metrics"
	"beacon-system/modules/advanced_analysis"
	"beacon-system/modules/alarm_mqtt"
	"beacon-system/modules/dtu_receiver"
	"beacon-system/modules/eventbus"
	"beacon-system/modules/network_reliability_analyzer"
	"beacon-system/modules/visibility_analyzer"
	"beacon-system/mqtt"
	"log"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
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
	advancedAnalyzer := advanced_analysis.NewAdvancedAnalyzer(database.GetDB(), eventbus.Get())

	dtuReceiver.Start()
	visibilityAnalyzer.Start()
	networkAnalyzer.Start()
	alarmModule.Start()
	defer alarmModule.Stop()

	handlers.InitDTUReceiver(dtuReceiver)
	handlers.InitVisibilityAnalyzer(visibilityAnalyzer)
	handlers.InitNetworkModules(networkAnalyzer, alarmModule)
	handlers.InitHandlers()
	advancedHandler := handlers.NewAdvancedHandler(advancedAnalyzer)

	log.Println("[Init] All modules initialized: dtu_receiver, visibility_analyzer, network_reliability_analyzer, alarm_mqtt, advanced_analysis")
	if params != nil {
		log.Printf("[Init] Params loaded: DEM radius=%dm, ITU-R k=%.3f, MC iterations=%d, IS edge threshold=%d",
			int(params.Terrain.DEMSearchRadiusMeters),
			params.Atmosphere.EffectiveEarthFactorK,
			params.Reliability.DefaultMCIterations,
			params.Reliability.ISEdgeThreshold)
	} else {
		log.Println("[Init] Params not available (using defaults embedded in modules)")
	}

	go startMetricsServer(cfg.ServerPort)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(metricsMiddleware())
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

		advancedHandler.RegisterRoutes(api)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":     "ok",
			"service":    "beacon-visibility-analysis-system",
			"version":    Version,
			"build_time": BuildTime,
			"modules":    []string{"dtu_receiver", "visibility_analyzer", "network_reliability_analyzer", "alarm_mqtt", "advanced_analysis"},
		})
	})

	r.GET("/params", func(c *gin.Context) {
		c.JSON(200, params)
	})

	log.Printf("Server starting on port %s... (version=%s)", cfg.ServerPort, Version)
	log.Printf("Demo mode: %v", cfg.DemoMode)
	log.Printf("Metrics: http://localhost:%s/metrics", cfg.ServerPort)
	log.Printf("pprof: http://localhost:6060/debug/pprof/")
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func startMetricsServer(appPort string) {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	log.Printf("Metrics & pprof server on :6060")
	if err := http.ListenAndServe(":6060", mux); err != nil {
		log.Printf("Metrics server error: %v", err)
	}
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		metrics.HttpRequests.WithLabelValues(c.Request.Method, path, statusLabel(status)).Inc()
		metrics.HttpDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

func statusLabel(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}
