package routes

import (
	"net/http"

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
	router := gin.Default()
	router.Use(middleware.RateLimiter())

	// ── Services ────────────────────────────────────────────────────────
	authService := &services.AuthService{DB: db}
	userService := &services.UserService{DB: db}
	recordService := &services.RecordService{DB: db}
	dashboardService := &services.DashboardService{DB: db}

	// ── Handlers ────────────────────────────────────────────────────────
	authHandler := &handlers.AuthHandler{Service: authService}
	userHandler := &handlers.UserHandler{Service: userService}
	recordHandler := &handlers.RecordHandler{Service: recordService}
	dashboardHandler := &handlers.DashboardHandler{Service: dashboardService}

	// ── Health check ────────────────────────────────────────────────────
	router.GET("/health", func(c *gin.Context) {
		utils.Success(c, http.StatusOK, "service is healthy", nil)
	})

	// ── Public routes (no auth required) ────────────────────────────────
	auth := router.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// ── Protected routes (JWT + RBAC) ───────────────────────────────────
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
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
	}

	return router
}
