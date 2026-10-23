package middlewear

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func TestSetJWTSecretRejectsShortSecret(t *testing.T) {
	if err := SetJWTSecret("ice_moss"); err == nil {
		t.Fatal("expected the exposed short secret to be rejected")
	}
}

func TestParseTokenRejectsUnexpectedAlgorithm(t *testing.T) {
	if err := SetJWTSecret(testJWTSecret); err != nil {
		t.Fatal(err)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, Claims{
		UserID: 1,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
	})
	signedToken, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ParseToken(signedToken); err == nil {
		t.Fatal("expected a non-HS256 token to be rejected")
	}
}

func TestJWYSetsAuthenticatedUserAndRejectsMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if err := SetJWTSecret(testJWTSecret); err != nil {
		t.Fatal(err)
	}
	token, err := GenerateToken(42, "test")
	if err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.GET("/protected", JWY(), func(c *gin.Context) {
		userID, ok := AuthenticatedUserID(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"userId": userID})
	})

	request := httptest.NewRequest(http.MethodGet, "/protected?userId=42", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/protected?userId=7&token="+token, nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected mismatched user ID to be rejected, got %d", response.Code)
	}
}
