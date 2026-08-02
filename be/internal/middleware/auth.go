package middleware

import (
	"github.com/anisharaz/incus-k8s-manager/be/internal/auth"
	"github.com/anisharaz/incus-k8s-manager/be/internal/models"
	"github.com/gofiber/fiber/v3"
)

// AuthCookieName is the HttpOnly cookie carrying the session JWT.
const AuthCookieName = "auth_token"

// authClaimsLocalsKey is the fiber.Ctx locals key RequireAuth stores the
// validated claims under, for RequireAdmin (or handlers) to read back.
const authClaimsLocalsKey = "authClaims"

// RequireAuth validates the session cookie and stashes its claims in
// locals for downstream handlers/middleware. Responds 401 if missing/invalid.
func RequireAuth(secret []byte) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := c.Cookies(AuthCookieName)
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
				Error:   "unauthorized",
				Message: "not logged in",
				Code:    fiber.StatusUnauthorized,
			})
		}

		claims, err := auth.ParseToken(secret, token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
				Error:   "unauthorized",
				Message: "session is invalid or expired, please log in again",
				Code:    fiber.StatusUnauthorized,
			})
		}

		c.Locals(authClaimsLocalsKey, claims)
		return c.Next()
	}
}

// RequireAdmin rejects the request unless RequireAuth ran first and the
// caller's role is admin.
func RequireAdmin(c fiber.Ctx) error {
	claims, _ := c.Locals(authClaimsLocalsKey).(*auth.Claims)
	if claims == nil || claims.Role != string(models.UserRoleAdmin) {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Error:   "forbidden",
			Message: "admin role required",
			Code:    fiber.StatusForbidden,
		})
	}

	return c.Next()
}

// ClaimsFromContext returns the claims RequireAuth stashed, or nil if
// RequireAuth didn't run (e.g. a route with no auth middleware).
func ClaimsFromContext(c fiber.Ctx) *auth.Claims {
	claims, _ := c.Locals(authClaimsLocalsKey).(*auth.Claims)
	return claims
}
