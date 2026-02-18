package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

var testSecret = []byte("test-secret-key-for-jwt-middleware-testing")

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
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
	r.Use(jwtMiddleware(testSecret, logger))
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
	r.Use(jwtMiddleware(testSecret, logger))

	// Register all public paths and prefixes.
	publicPaths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/auth/login"},
		{http.MethodPost, "/api/auth/register"},
		{http.MethodPost, "/api/auth/logout"},
		{http.MethodGet, "/health"},
		{http.MethodGet, "/api/attachments/123"},
		{http.MethodGet, "/api/attachments/some-file.jpg"},
	}

	for _, pp := range publicPaths {
		switch pp.method {
		case http.MethodGet:
			r.GET(pp.path, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
		case http.MethodPost:
			r.POST(pp.path, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
		}
	}

	for _, pp := range publicPaths {
		t.Run(pp.method+" "+pp.path, func(t *testing.T) {
			req := httptest.NewRequest(pp.method, pp.path, http.NoBody)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}
		})
	}
}

func TestJWTMiddleware_MalformedAuthorizationHeader(t *testing.T) {
	r := setupMiddlewareTest()

	tests := []struct {
		name   string
		header string
	}{
		{"empty header", ""},
		{"no bearer prefix", "Token abc123"},
		{"bearer only", "Bearer "},
		{"lowercase bearer with garbage", "bearer not-a-jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestJWTMiddleware_WebSocketQueryToken(t *testing.T) {
	r := gin.New()
	logger := zap.NewNop()
	r.Use(jwtMiddleware(testSecret, logger))
	r.GET("/ws", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not set"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	token := createTestToken(t, validClaims(7), testSecret)

	t.Run("valid token in query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws?token="+token, http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}
	})

	t.Run("missing token in query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ws", http.NoBody)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})
}
