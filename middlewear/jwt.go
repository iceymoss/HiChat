package middlewear

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

var (
	TokenExpired         = errors.New("token is expired")
	ErrInvalidToken      = errors.New("invalid token")
	ErrJWTSecretTooShort = errors.New("JWT secret must contain at least 32 bytes")
)

const userIDContextKey = "authenticatedUserID"

var jwtSecret []byte

func SetJWTSecret(secret string) error {
	if len(secret) < 32 {
		return ErrJWTSecretTooShort
	}
	jwtSecret = []byte(secret)
	return nil
}

// Claims 是一些实体（通常指的用户）的状态和额外的元数据
type Claims struct {
	UserID uint `json:"userId"`
	jwt.StandardClaims
}

func JWY() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if authorization := c.GetHeader("Authorization"); strings.HasPrefix(authorization, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, map[string]string{
				"message": "请登录",
			})
			c.Abort()
			return
		}

		claims, err := ParseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, map[string]string{
				"message": "token失效",
			})
			c.Abort()
			return
		}
		if time.Now().Unix() > claims.ExpiresAt {
			c.JSON(http.StatusUnauthorized, map[string]string{
				"message": "授权已过期",
			})
			c.Abort()
			return
		}

		if user := c.Query("userId"); user != "" {
			userID, err := strconv.ParseUint(user, 10, 64)
			if err != nil || claims.UserID != uint(userID) {
				c.JSON(http.StatusUnauthorized, map[string]string{
					"message": "您的登录不合法",
				})
				c.Abort()
				return
			}
		}

		c.Set(userIDContextKey, claims.UserID)
		c.Next()
	}
}

func AuthenticatedUserID(c *gin.Context) (uint, bool) {
	userID, ok := c.Get(userIDContextKey)
	if !ok {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}

// GenerateToken 根据用户的用户名和密码产生token
func GenerateToken(userId uint, iss string) (string, error) {
	if len(jwtSecret) < 32 {
		return "", ErrJWTSecretTooShort
	}
	//设置token有效时间
	nowTime := time.Now()
	expireTime := nowTime.Add(24 * time.Hour)

	claims := Claims{
		UserID: userId,
		StandardClaims: jwt.StandardClaims{
			// 过期时间
			ExpiresAt: expireTime.Unix(),
			// 指定token发行人
			Issuer: iss,
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	//该方法内部生成签名字符串，再用于获取完整、已签名的token
	token, err := tokenClaims.SignedString(jwtSecret)
	return token, err
}

// ParseToken 根据传入的token值获取到Claims对象信息（进而获取其中的用户id）
func ParseToken(token string) (*Claims, error) {
	if len(jwtSecret) < 32 {
		return nil, ErrJWTSecretTooShort
	}
	//用于解析鉴权的声明，方法内部主要是具体的解码和校验的过程，最终返回*Token
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return jwtSecret, nil
	})

	if tokenClaims != nil {
		// 从tokenClaims中获取到Claims对象，并使用断言，将该对象转换为我们自己定义的Claims
		// 要传入指针，项目中结构体都是用指针传递，节省空间。
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}
	if err == nil {
		err = ErrInvalidToken
	}
	return nil, err
}
