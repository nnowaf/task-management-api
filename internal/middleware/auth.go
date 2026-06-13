package middleware

import (
	"strings"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/pkg/jwt"
	"github.com/gofiber/fiber/v2"
)

// NewAuth validates the access token and injects the user identity into the
// request locals (and the request-scoped logger).
func NewAuth(jwtMgr *jwt.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := strings.TrimSpace(c.Get("Authorization"))
		if len(token) > 7 && strings.EqualFold(token[:7], "Bearer ") {
			token = strings.TrimSpace(token[7:])
		}
		if token == "" {
			return domain.NewUnauthorized("missing Authorization header")
		}

		claims, err := jwtMgr.Parse(token)
		if err != nil {
			return domain.NewUnauthorized("invalid or expired token")
		}

		c.Locals(localUserID, claims.UserID)
		c.Locals(localUsername, claims.Username)

		l := Logger(c).With().Str("user_id", claims.UserID.String()).Logger()
		c.Locals(localLogger, &l)
		return c.Next()
	}
}
