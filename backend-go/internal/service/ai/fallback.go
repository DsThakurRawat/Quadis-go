package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"quadis-backend-go/internal/domain"
	"quadis-backend-go/internal/repository"
)

const ContactNumber = "+91 92173 73532"

type FallbackEngine struct {
	repo repository.Repository
}

func NewFallbackEngine(repo repository.Repository) *FallbackEngine {
	return &FallbackEngine{repo: repo}
}

func (f *FallbackEngine) GenerateFallbackResponse(ctx context.Context, message string) ChatResponse {
	lower := strings.ToLower(strings.TrimSpace(message))

	// 1. Human handoff detection
	handoffKeywords := []string{
		"manager", "call me", "human", "complaint", "talk to someone",
		"speak to someone", "agent", "reception", "customer care", "person", "refund",
	}
	for _, kw := range handoffKeywords {
		if strings.Contains(lower, kw) {
			return ChatResponse{
				Response: fmt.Sprintf(
					"I understand you'd like to speak with our team directly. 🛎️ Please call or WhatsApp our front desk manager at %s, or share your phone number and we'll reach out immediately.",
					ContactNumber,
				),
				HandoffTriggered: true,
				ProviderUsed:     "deterministic-fallback",
			}
		}
	}

	// 2. Booking / Hold requests
	if strings.Contains(lower, "hold") || strings.Contains(lower, "reserve") || strings.Contains(lower, "book") {
		return ChatResponse{
			Response: fmt.Sprintf(
				"Happy to help! 🛎️ To ensure your dates and room preferences are locked in with instant confirmation, please select your dates and room directly on the site, or call our reservations desk at %s.",
				ContactNumber,
			),
			ProviderUsed: "deterministic-fallback",
		}
	}

	// 3. Location / Address / Directions
	if strings.Contains(lower, "where") || strings.Contains(lower, "location") || strings.Contains(lower, "address") ||
		strings.Contains(lower, "airport") || strings.Contains(lower, "metro") || strings.Contains(lower, "near") ||
		strings.Contains(lower, "map") {

		props, _ := f.repo.GetProperties(ctx)
		var pool []domain.PropertyRecord
		wantedCity := ""
		if strings.Contains(lower, "delhi") {
			wantedCity = "New Delhi"
		} else if strings.Contains(lower, "noida") {
			wantedCity = "Noida"
		}

		for _, p := range props {
			if wantedCity == "" || strings.EqualFold(string(p.City), wantedCity) {
				pool = append(pool, p)
			}
		}

		if len(pool) > 0 {
			if len(pool) > 4 {
				pool = pool[:4]
			}
			var lines []string
			for _, p := range pool {
				line := fmt.Sprintf("• *%s* — %s", p.Name, p.Address)
				if p.MapLink != nil && *p.MapLink != "" {
					line += fmt.Sprintf("\n  Map: %s", *p.MapLink)
				}
				lines = append(lines, line)
			}
			header := "Our locations"
			if wantedCity != "" {
				header = fmt.Sprintf("Our locations in %s", wantedCity)
			}
			resp := fmt.Sprintf("%s:\n\n%s\n\nFor directions or landmark details, feel free to call %s.", header, strings.Join(lines, "\n"), ContactNumber)
			return ChatResponse{
				Response:     resp,
				ProviderUsed: "deterministic-fallback",
				ToolsInvoked: []string{"get_locations"},
			}
		}
	}

	// 4. Meals
	if strings.Contains(lower, "breakfast") || strings.Contains(lower, "meal") || strings.Contains(lower, "food") ||
		strings.Contains(lower, "dinner") || strings.Contains(lower, "lunch") {
		return ChatResponse{
			Response: fmt.Sprintf(
				"Rooms are Room Only as standard. We offer:\n• *CP (With Breakfast)*: adds 25%% to the nightly room rate\n• *MAP (All Meals Included)*: adds 50%% to the nightly room rate\n\nThese percentage uplifts apply consistently across all our hotels in Delhi & Noida. You can select your preferred meal plan during checkout on the site, or call %s.",
				ContactNumber,
			),
			ProviderUsed: "deterministic-fallback",
		}
	}

	// 5. Amenities
	if strings.Contains(lower, "wifi") || strings.Contains(lower, "wi-fi") || strings.Contains(lower, "parking") ||
		strings.Contains(lower, "pool") || strings.Contains(lower, "gym") || strings.Contains(lower, "amenit") ||
		strings.Contains(lower, "ac") {
		return ChatResponse{
			Response: fmt.Sprintf(
				"All Quadis properties feature complimentary high-speed Wi-Fi, 24/7 room service, clean linens, daily housekeeping, and air-conditioned rooms. For property-specific amenities like elevator access or on-site parking, please WhatsApp %s.",
				ContactNumber,
			),
			ProviderUsed: "deterministic-fallback",
		}
	}

	// 6. Banquets
	if strings.Contains(lower, "banquet") || strings.Contains(lower, "wedding") || strings.Contains(lower, "event") ||
		strings.Contains(lower, "party") {
		return ChatResponse{
			Response: fmt.Sprintf(
				"We host weddings, corporate conferences, and social celebrations with full catering, audiovisual setups, and parking at our banquet venues including Amaltas International (Green Park Extension) and Sector 51 Noida. Please call %s or submit a banquet enquiry on the site for date availability and tailored pricing.",
				ContactNumber,
			),
			ProviderUsed: "deterministic-fallback",
		}
	}

	// 7. Rooms, Rates & Availability
	props, _ := f.repo.GetProperties(ctx)
	if len(props) > 0 {
		sort.Slice(props, func(i, j int) bool {
			return props[i].BasePrice < props[j].BasePrice
		})
		var lines []string
		for i, p := range props {
			if i >= 5 {
				break
			}
			lines = append(lines, fmt.Sprintf("• %s (%s) — from ₹%.0f/night", p.Name, p.City, p.BasePrice))
		}
		return ChatResponse{
			Response: fmt.Sprintf(
				"Welcome to Quadis Hotels! 🏨 We offer premium stays across New Delhi and Noida:\n\n%s\n\nWhich dates are you planning to visit? You can also reach our reservations desk at %s.",
				strings.Join(lines, "\n"),
				ContactNumber,
			),
			ProviderUsed: "deterministic-fallback",
			ToolsInvoked: []string{"search_hotels"},
		}
	}

	// Default greeting
	return ChatResponse{
		Response: fmt.Sprintf(
			"Welcome to Quadis Hotels! 🏨 How can I assist you with your stay today? Feel free to ask about our properties in Delhi & Noida, room rates, meal plans, or reach us at %s.",
			ContactNumber,
		),
		ProviderUsed: "deterministic-fallback",
	}
}
