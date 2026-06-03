package routes

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"finance-dashboard/handlers"
	"finance-dashboard/middleware"
	"finance-dashboard/services"
	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRoutes creates a Gin engine, initialises all services and handlers,
// and registers every route with the appropriate middleware.
func SetupRoutes(db *gorm.DB) *gin.Engine {
	// Set Gin mode based on environment (R3: production should use release mode).
	if os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// ── Global middleware (order matters) ────────────────────────────────
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogger())
	router.Use(middleware.RateLimiter())

	// ── CORS middleware (S7: allow cross-origin requests) ────────────────
	router.Use(corsMiddleware())

	// ── Token expiry config (from env, with sensible defaults) ───────────
	accessExpiryMins := 15
	refreshExpiryDays := 7
	if v := os.Getenv("ACCESS_TOKEN_EXPIRY_MINUTES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			accessExpiryMins = parsed
		}
	}
	if v := os.Getenv("REFRESH_TOKEN_EXPIRY_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			refreshExpiryDays = parsed
		}
	}

	jwtSecret := os.Getenv("JWT_SECRET")

	// ── Services ────────────────────────────────────────────────────────
	tokenService := &services.TokenService{
		DB:                     db,
		JWTSecret:              jwtSecret,
		AccessTokenExpiryMins:  accessExpiryMins,
		RefreshTokenExpiryDays: refreshExpiryDays,
	}
	accountService := &services.AccountService{DB: db}
	auditService := &services.AuditService{DB: db}

	authService := &services.AuthService{
		DB:             db,
		TokenService:   tokenService,
		AccountService: accountService,
		AuditService:   auditService,
	}
	userService := &services.UserService{
		DB:           db,
		AuditService: auditService,
	}
	recordService := &services.RecordService{
		DB:           db,
		AuditService: auditService,
	}
	dashboardService := &services.DashboardService{DB: db}
	ledgerService := &services.LedgerService{
		DB:           db,
		AuditService: auditService, // C10: ledger now has audit trail
	}

	// ── Handlers ────────────────────────────────────────────────────────
	authHandler := &handlers.AuthHandler{Service: authService}
	userHandler := &handlers.UserHandler{Service: userService}
	recordHandler := &handlers.RecordHandler{Service: recordService}
	dashboardHandler := &handlers.DashboardHandler{Service: dashboardService}
	ledgerHandler := &handlers.LedgerHandler{
		LedgerService:  ledgerService,
		AccountService: accountService,
	}
	auditHandler := &handlers.AuditHandler{Service: auditService}
	tokenHandler := &handlers.TokenHandler{TokenService: tokenService}

	// ── Health check (R4: verify database connectivity) ─────────────────
	router.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			utils.Error(c, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		if err := sqlDB.Ping(); err != nil {
			utils.Error(c, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		utils.Success(c, http.StatusOK, "service is healthy", map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
		})
	})

	// ── Public routes (no auth required) ────────────────────────────────
	auth := router.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", tokenHandler.Refresh)
	}

	// ── Protected routes (JWT + RBAC) ───────────────────────────────────
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware(jwtSecret)) // S2: pass secret explicitly
	api.Use(middleware.IdempotencyMiddleware(db))
	{
		// Auth — logout requires authentication
		api.POST("/auth/logout", tokenHandler.Logout)

		// User management — admin only
		users := api.Group("/users")
		{
			users.GET("", middleware.RequireRole("admin"), userHandler.GetUsers)
			users.PUT("/:id", middleware.RequireRole("admin"), userHandler.UpdateUser)
			users.DELETE("/:id", middleware.RequireRole("admin"), userHandler.DeleteUser)
		}

		// Financial records
		records := api.Group("/records")
		{
			records.GET("", middleware.RequireRole("viewer", "analyst", "admin"), recordHandler.GetRecords)
			records.POST("", middleware.RequireRole("admin"), recordHandler.CreateRecord)
			records.GET("/:id", middleware.RequireRole("viewer", "analyst", "admin"), recordHandler.GetRecordByID)
			records.PUT("/:id", middleware.RequireRole("admin"), recordHandler.UpdateRecord)
			records.DELETE("/:id", middleware.RequireRole("admin"), recordHandler.DeleteRecord)
		}

		// Dashboard analytics
		dashboard := api.Group("/dashboard")
		{
			dashboard.GET("/summary", middleware.RequireRole("viewer", "analyst", "admin"), dashboardHandler.GetSummary)
			dashboard.GET("/trends", middleware.RequireRole("analyst", "admin"), dashboardHandler.GetTrends)
			dashboard.GET("/categories", middleware.RequireRole("analyst", "admin"), dashboardHandler.GetCategoryBreakdown)
		}

		// Double-entry ledger
		ledger := api.Group("/ledger")
		{
			ledger.POST("/transactions", middleware.RequireRole("admin"), ledgerHandler.PostTransaction)
			ledger.GET("/transactions", middleware.RequireRole("viewer", "analyst", "admin"), ledgerHandler.GetTransactions)
			ledger.GET("/accounts", middleware.RequireRole("viewer", "analyst", "admin"), ledgerHandler.GetAccounts)
			ledger.POST("/accounts", middleware.RequireRole("admin"), ledgerHandler.CreateAccount)
			ledger.GET("/accounts/:id/entries", middleware.RequireRole("viewer", "analyst", "admin"), ledgerHandler.GetAccountEntries)
		}

		// Audit log — admin only
		api.GET("/audit", middleware.RequireRole("admin"), auditHandler.GetAuditLog)
	}

	return router
}

// corsMiddleware adds Cross-Origin Resource Sharing headers (S7).
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-ID")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
