package api

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// AuthMiddleware handles API authentication
type AuthMiddleware struct {
	jwtSecret []byte
	apiKeys   map[string]*APIKey
	logger    *logger.Logger
}

// APIKey represents an API key with associated permissions
type APIKey struct {
	Key         string
	Name        string
	Roles       []string
	RateLimitPerMin int
	CreatedAt   time.Time
	LastUsed    *time.Time
}

// Claims represents JWT claims
type Claims struct {
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtSecret string, logger *logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: []byte(jwtSecret),
		apiKeys:   make(map[string]*APIKey),
		logger:    logger,
	}
}

// AddAPIKey adds an API key
func (a *AuthMiddleware) AddAPIKey(key, name string, roles []string) {
	a.apiKeys[key] = &APIKey{
		Key:         key,
		Name:        name,
		Roles:       roles,
		RateLimitPerMin: 60,
		CreatedAt:   time.Now(),
	}
}

// Authenticate returns a middleware function for authentication
func (a *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for Bearer token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if a.validateJWT(c, token) {
					c.Next()
					return
				}
			}
		}

		// Check for API key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			if a.validateAPIKey(c, apiKey) {
				c.Next()
				return
			}
		}

		// No valid authentication
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		c.Abort()
	}
}

// RequireRole returns a middleware that requires specific roles
func (a *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "No roles found",
			})
			c.Abort()
			return
		}

		userRoleList := userRoles.([]string)
		for _, requiredRole := range roles {
			for _, userRole := range userRoleList {
				if userRole == requiredRole || userRole == "admin" {
					c.Next()
					return
				}
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions",
		})
		c.Abort()
	}
}

// validateJWT validates a JWT token
func (a *AuthMiddleware) validateJWT(c *gin.Context, tokenString string) bool {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		a.logger.Debug("JWT validation failed", "error", err.Error())
		return false
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		return true
	}

	return false
}

// validateAPIKey validates an API key
func (a *AuthMiddleware) validateAPIKey(c *gin.Context, key string) bool {
	apiKey, exists := a.apiKeys[key]
	if !exists {
		a.logger.Debug("API key not found", "key", key[:8]+"...")
		return false
	}

	// Update last used time
	now := time.Now()
	apiKey.LastUsed = &now

	// Set context values
	c.Set("api_key_name", apiKey.Name)
	c.Set("roles", apiKey.Roles)

	return true
}

// GenerateJWT generates a new JWT token
func (a *AuthMiddleware) GenerateJWT(username string, roles []string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "wemixvisor",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// RateLimiter returns a middleware for rate limiting
func RateLimiter(requestsPerMinute int) gin.HandlerFunc {
	// Simplified rate limiter for demo
	// In production, use a proper rate limiting library
	requests := make(map[string][]time.Time)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		minute := now.Add(-time.Minute)

		// Clean old entries
		if reqs, exists := requests[ip]; exists {
			var filtered []time.Time
			for _, t := range reqs {
				if t.After(minute) {
					filtered = append(filtered, t)
				}
			}
			requests[ip] = filtered
		}

		// Check rate limit
		if len(requests[ip]) >= requestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		// Add current request
		requests[ip] = append(requests[ip], now)
		c.Next()
	}
}

// BasicAuth returns a middleware for basic authentication
func BasicAuth(username, password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, pass, hasAuth := c.Request.BasicAuth()
		if !hasAuth {
			c.Header("WWW-Authenticate", `Basic realm="Wemixvisor API"`)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Basic authentication required",
			})
			c.Abort()
			return
		}

		// Use constant time comparison to prevent timing attacks
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1

		if !userMatch || !passMatch {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid credentials",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}