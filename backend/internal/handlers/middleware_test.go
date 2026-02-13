package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

var testSecret = []byte("test-secret-key-for-jwt-middleware-testing")

func init() {
	gin.SetMode(gin.TestMode)
}

// createTestToken generates a signed JWT with the given claims and secret.
func createTestToken(t *testing.T, claims jwt.MapClaims, secret []byte) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return signed
}

// validClaims returns standard valid claims for testing.
func validClaims(userID float64) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"user_id": userID,
		"iss":     "messenger-app",
		"aud":     "messenger-users",
		"exp":     now.Add(time.Hour).Unix(),
		"iat":     now.Unix(),
	}
}

// setupMiddlewareTest creates a Gin engine with the JWT middleware and a test handler.
// The test handler returns 200 with the user_id from context.
func setupMiddlewareTest() *gin.Engine {
	r := gin.New()
	logger := zap.NewNop()
	r.Use(JWTMiddleware(testSecret, logger))
	r.GET("/api/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not set"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	r := setupMiddlewareTest()
	token := createTestToken(t, validClaims(42), testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	r := setupMiddlewareTest()

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	r := setupMiddlewareTest()

	claims := validClaims(42)
	claims["exp"] = time.Now().Add(-time.Hour).Unix() // expired
	token := createTestToken(t, claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_InvalidSignature(t *testing.T) {
	r := setupMiddlewareTest()

	// Sign with a different secret
	wrongSecret := []byte("wrong-secret-key-that-does-not-match!!")
	token := createTestToken(t, validClaims(42), wrongSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_WrongIssuer(t *testing.T) {
	r := setupMiddlewareTest()

	claims := validClaims(42)
	claims["iss"] = "wrong-issuer"
	token := createTestToken(t, claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_WrongAudience(t *testing.T) {
	r := setupMiddlewareTest()

	claims := validClaims(42)
	claims["aud"] = "wrong-audience"
	token := createTestToken(t, claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_MissingUserIDClaim(t *testing.T) {
	r := setupMiddlewareTest()

	claims := validClaims(42)
	delete(claims, "user_id")
	token := createTestToken(t, claims, testSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_PublicPathSkipped(t *testing.T) {
	r := gin.New()
	logger := zap.NewNop()
	r.Use(JWTMiddleware(testSecret, logger))
	r.POST("/api/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/api/auth/register", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// No token needed for public paths
	for _, path := range []string{"/api/auth/login", "/api/auth/register"} {
		req := httptest.NewRequest(http.MethodPost, path, http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("path %s: status = %d, want %d", path, w.Code, http.StatusOK)
		}
	}
}
