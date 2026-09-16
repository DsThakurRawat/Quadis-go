package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"quadis-backend-go/internal/api/handlers"
	"quadis-backend-go/internal/api/middleware"
	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/repository"
	"quadis-backend-go/internal/service/ai"
	"quadis-backend-go/internal/service/auth"
	"quadis-backend-go/internal/service/booking"
	"quadis-backend-go/internal/service/ota"
	"quadis-backend-go/internal/service/payment"
)

type RouterDeps struct {
	Config          *config.Config
	Repo            repository.Repository
	SessionService  *auth.SessionService
	BookingService  *booking.BookingService
	RazorpayService *payment.RazorpayService
	OTAService      *ota.OTAService
	AIService       *ai.AIService
}

// NewRouter constructs the Chi HTTP router with all middlewares and route handlers.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// 1. Core Global Middlewares
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders(deps.Config.IsProduction()))
	r.Use(middleware.NewCORS(deps.Config.CORSOrigins, deps.Config.IsProduction()))
	r.Use(middleware.PreserveRawBody(2 * 1024 * 1024)) // 2MB body limit matching Express /api/ota

	// Rate limiters
	apiLimiter := middleware.NewIPRateLimiter(100, 15*time.Minute, deps.Config.IsProduction(), nil)
	authLimiter := middleware.NewIPRateLimiter(10, 15*time.Minute, deps.Config.IsProduction(),
		[]byte(`{"success":false,"error":"Too many sign-in attempts. Try again later."}`),
	)
	otaLimiter := middleware.NewIPRateLimiter(1000, 15*time.Minute, deps.Config.IsProduction(),
		[]byte(`{"OTA_ErrorRS":{"Status":"Failure","Remark":"Too many requests. Try again later."}}`),
	)
	cspLimiter := middleware.NewIPRateLimiter(30, time.Minute, deps.Config.IsProduction(), nil)

	// Auth middleware
	authGuard := middleware.NewAuthMiddleware(deps.SessionService, deps.Config)

	// Handlers
	healthH := handlers.NewHealthHandler(deps.Repo, deps.Config)
	propsH := handlers.NewPropertiesHandler(deps.Repo)
	contentH := handlers.NewContentHandler(deps.Repo)
	bookingsH := handlers.NewBookingsHandler(deps.Repo, deps.BookingService, deps.SessionService, deps.Config)
	paymentsH := handlers.NewPaymentsHandler(deps.Repo, deps.RazorpayService)
	webhooksH := handlers.NewWebhooksHandler(deps.Repo, deps.RazorpayService, deps.OTAService, deps.Config)
	enquiriesH := handlers.NewEnquiriesHandler(deps.Repo)
	authH := handlers.NewAuthHandler(deps.Repo, deps.SessionService, deps.Config)
	adminH := handlers.NewAdminHandler(deps.Repo, deps.SessionService, deps.RazorpayService, deps.Config)
	aiH := handlers.NewAIHandler(deps.AIService, deps.Repo)
	otaH := handlers.NewOTAHandler(deps.Repo, deps.OTAService, deps.Config)

	// Lightweight LB health check
	r.Get("/healthz", healthH.Healthz)

	// API Routes Root
	r.Route("/api", func(api chi.Router) {
		// Health
		api.Get("/health", healthH.Health)

		// CSP Report
		api.With(cspLimiter.Handler).Post("/csp-report", handlers.CSPReport)

		// Properties
		api.Route("/properties", func(pr chi.Router) {
			pr.Get("/", propsH.ListProperties)
			pr.Get("/{slug}", propsH.GetPropertyBySlug)
		})

		// Content
		api.Get("/content", contentH.GetContent)

		// Bookings
		api.Route("/bookings", func(br chi.Router) {
			br.With(apiLimiter.Handler).Post("/initiate", bookingsH.InitiateBooking)
			br.With(apiLimiter.Handler).Get("/{code}", bookingsH.GetBookingByCode)
			br.Get("/{code}/invoice", bookingsH.GetInvoice)
		})

		// Payments
		api.Route("/payments", func(pay chi.Router) {
			pay.Post("/create-order", paymentsH.CreateOrder)
			pay.Post("/payment-link", paymentsH.PaymentLink)
			pay.With(authGuard.RequireAdmin).Post("/enquiry-payment-link", paymentsH.EnquiryPaymentLink)
		})

		// Webhooks
		api.Route("/webhooks", func(wh chi.Router) {
			wh.Post("/razorpay", webhooksH.RazorpayWebhook)
			wh.Post("/whatsapp-staff", webhooksH.WhatsAppStaff)
		})

		// Enquiries
		api.Route("/enquiries", func(er chi.Router) {
			er.With(apiLimiter.Handler).Post("/", enquiriesH.CreateEnquiry)
			er.With(authGuard.RequireAdmin).Get("/", enquiriesH.GetEnquiries)
			er.With(authGuard.RequireAdmin).Get("/{id}", enquiriesH.GetEnquiryByID)
			er.With(authGuard.RequireAdmin).Patch("/{id}/status", enquiriesH.UpdateEnquiryStatus)
		})

		// Auth
		api.Route("/auth", func(ar chi.Router) {
			ar.With(authLimiter.Handler).Post("/register", authH.Register)
			ar.With(authLimiter.Handler).Post("/login", authH.Login)
			ar.Get("/me", authH.Me)
			ar.With(authGuard.RequireUser).Get("/bookings", authH.Bookings)
		})

		// Admin
		api.Route("/admin", func(adm chi.Router) {
			// Unprotected PIN auth
			adm.With(authLimiter.Handler).Post("/auth", adminH.Auth)

			// Protected Admin Endpoints
			adm.Group(func(guarded chi.Router) {
				guarded.Use(authGuard.RequireAdmin)

				guarded.Post("/change-pin", adminH.ChangePIN)
				guarded.Get("/dashboard", adminH.Dashboard)
				guarded.Patch("/room-availability", adminH.RoomAvailability)
				guarded.Patch("/properties/{idOrSlug}", adminH.UpdateProperty)
				guarded.Patch("/room-types/{id}", adminH.UpdateRoomType)
				guarded.Put("/content", adminH.UpdateContent)
				guarded.Patch("/surcharge", adminH.Surcharge)
				guarded.Post("/payment-link", adminH.PaymentLink)
				guarded.Patch("/properties/{idOrSlug}/images/order", adminH.ReorderImages)
				guarded.Delete("/images/{id}", adminH.DeleteImage)
			})
		})

		// AI Concierge
		api.Route("/ai", func(air chi.Router) {
			air.With(apiLimiter.Handler).Post("/chat", aiH.Chat)
			air.With(authGuard.RequireAdmin).Get("/logs", aiH.Logs)
		})

		// ResAvenue OTA Channel Manager
		api.Route("/ota", func(otar chi.Router) {
			otar.Use(otaLimiter.Handler)

			// GET /codes
			otar.Get("/codes", otaH.Codes)

			// Specific routes (support POST and PUT)
			otar.HandleFunc("/property-details", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost || r.Method == http.MethodPut {
					otaH.PropertyDetails(w, r)
					return
				}
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			})

			otar.HandleFunc("/inventory/update", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost || r.Method == http.MethodPut {
					otaH.InventoryUpdate(w, r)
					return
				}
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			})

			otar.HandleFunc("/rates/update", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost || r.Method == http.MethodPut {
					otaH.RateUpdate(w, r)
					return
				}
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			})

			otar.HandleFunc("/bookings/pull", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost || r.Method == http.MethodPut {
					otaH.BookingPull(w, r)
					return
				}
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			})

			// Generic dispatcher routes
			otar.HandleFunc("/resavenue", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost || r.Method == http.MethodPut {
					otaH.Dispatch(w, r)
					return
				}
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			})

			otar.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost || r.Method == http.MethodPut {
					otaH.Dispatch(w, r)
					return
				}
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			})
		})
	})

	// Global 404 Fallback
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		handlers.ErrorJSON(w, http.StatusNotFound, "Endpoint not found")
	})

	return r
}
