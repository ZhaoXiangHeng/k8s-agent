// Package auth 提供 HTTP 请求认证。支持三种模式：
//
//	dev  — 开发模式，从 X-Demo-User 头读取用户名，默认 "demo"/"operator"
//	jwt  — 生产模式，验证 Keycloak RS256 JWT，失败直接返回 401
//	none — 匿名模式，不验证身份（用于健康检查端点或内部服务调用）
//
// 通过 AUTH_MODE 环境变量选择，默认 "dev"。
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var pkgLog = logrus.WithField("component", "backend-api/auth")

// Mode 表示认证模式。
type Mode string

const (
	ModeDev  Mode = "dev"  // 开发模式：X-Demo-User 头 + 默认演示用户
	ModeJWT  Mode = "jwt"  // 生产模式：必须提供有效 Keycloak JWT
	ModeNone Mode = "none" // 匿名模式：不验证身份
)

// UserContext 保存认证后的用户信息，通过 context 在请求链中传递。
type UserContext struct {
	UserID    string
	Username  string
	Email     string
	Role      string
	RequestID string
}

type contextKey string

const userContextKey contextKey = "userContext"

// JWTValidator 处理所有认证模式的用户身份解析。
type JWTValidator struct {
	mode    Mode
	issuer  string
	jwksURL string
	client  *http.Client

	mu        sync.RWMutex
	publicKey *rsa.PublicKey
}

// NewJWTValidator 创建认证验证器。
//   - mode=dev: 无需其他参数
//   - mode=jwt: issuer 必填（Keycloak realm URL）
//   - mode=none: 无需其他参数
func NewJWTValidator(mode Mode, issuer string) *JWTValidator {
	if mode == ModeJWT && issuer == "" {
		pkgLog.WithField("event", "jwt_missing_issuer").Fatal("AUTH_MODE=jwt requires KEYCLOAK_ISSUER to be set")
	}
	v := &JWTValidator{
		mode:   mode,
		issuer: issuer,
		client: &http.Client{Timeout: 10 * time.Second},
	}
	if issuer != "" {
		v.jwksURL = strings.TrimRight(issuer, "/") + "/protocol/openid-connect/certs"
	}
	return v
}

// Middleware 返回 HTTP 中间件。
func (v *JWTValidator) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 健康检查端点跳过认证
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}
			w.Header().Set("X-Request-ID", requestID)

			user, err := v.resolve(r, requestID)
			if err != nil {
				pkgLog.WithError(err).WithFields(logrus.Fields{
					"event":      "auth_denied",
					"mode":       v.mode,
					"request_id": requestID,
					"path":       r.URL.Path,
				}).Warn("authentication denied")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintf(w, `{"error":{"code":"UNAUTHENTICATED","message":%q,"requestId":%q}}`, err.Error(), requestID)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// resolve 根据认证模式解析用户身份。
func (v *JWTValidator) resolve(r *http.Request, requestID string) (*UserContext, error) {
	switch v.mode {
	case ModeJWT:
		return v.resolveJWT(r, requestID)
	case ModeDev:
		return v.resolveDev(r, requestID), nil
	case ModeNone:
		return &UserContext{UserID: "anonymous", Username: "anonymous", RequestID: requestID}, nil
	default:
		return nil, fmt.Errorf("unknown auth mode: %s", v.mode)
	}
}

// resolveDev 开发模式：从请求头或默认值获取用户身份。
func (v *JWTValidator) resolveDev(r *http.Request, requestID string) *UserContext {
	username := r.Header.Get("X-Demo-User")
	if username == "" {
		username = "demo"
	}
	role := r.Header.Get("X-Demo-Role")
	if role == "" {
		role = "operator"
	}
	return &UserContext{
		UserID:    "user-" + username,
		Username:  username,
		Email:     username + "@demo.local",
		Role:      role,
		RequestID: requestID,
	}
}

// resolveJWT 生产模式：验证 Authorization: Bearer <token> 中的 JWT。
func (v *JWTValidator) resolveJWT(r *http.Request, requestID string) (*UserContext, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == authHeader {
		return nil, fmt.Errorf("invalid Authorization header format, expected 'Bearer <token>'")
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return v.getPublicKey()
	}, jwt.WithIssuer(v.issuer), jwt.WithLeeway(30*time.Second))

	if err != nil {
		return nil, fmt.Errorf("jwt validation failed: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &UserContext{
		UserID:    extractString(claims, "sub"),
		Username:  extractUsername(claims),
		Email:     extractString(claims, "email"),
		Role:      extractRole(claims),
		RequestID: requestID,
	}, nil
}

func extractUsername(claims jwt.MapClaims) string {
	if u := extractString(claims, "preferred_username"); u != "" {
		return u
	}
	return extractString(claims, "sub")
}

func extractRole(claims jwt.MapClaims) string {
	if role := extractString(claims, "role"); role != "" {
		return role
	}
	realmAccess, ok := claims["realm_access"].(map[string]interface{})
	if !ok {
		return ""
	}
	roles, ok := realmAccess["roles"].([]interface{})
	if !ok {
		return ""
	}
	for _, raw := range roles {
		if role, ok := raw.(string); ok && role == "admin" {
			return "admin"
		}
	}
	for _, raw := range roles {
		if role, ok := raw.(string); ok && role == "operator" {
			return "operator"
		}
	}
	return ""
}

// getPublicKey 获取 Keycloak JWKS 公钥（带双重检查锁缓存）。
func (v *JWTValidator) getPublicKey() (*rsa.PublicKey, error) {
	v.mu.RLock()
	if v.publicKey != nil {
		key := v.publicKey
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	v.mu.Lock()
	defer v.mu.Unlock()
	if v.publicKey != nil {
		return v.publicKey, nil
	}

	resp, err := v.client.Get(v.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks from %s: %w", v.jwksURL, err)
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}
	if len(jwks.Keys) == 0 {
		return nil, fmt.Errorf("no keys in JWKS response")
	}

	key := jwks.Keys[0]
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("decode RSA modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("decode RSA exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 65537 // 默认值
	if len(eBytes) <= 4 {
		e = int(new(big.Int).SetBytes(eBytes).Int64())
	}

	v.publicKey = &rsa.PublicKey{N: n, E: e}
	pkgLog.WithField("event", "jwks_cached").Info("Keycloak RSA public key cached")
	return v.publicKey, nil
}

// GetUserContext 从 context 中读取认证后的用户信息。
// 仅在 auth middleware 包裹后可用，未经认证的请求返回空用户。
func GetUserContext(ctx context.Context) *UserContext {
	user, _ := ctx.Value(userContextKey).(*UserContext)
	return user
}

// ResolveForGin 与 resolve 相同，但接受 *http.Request，供 Gin 中间件使用。
func (v *JWTValidator) ResolveForGin(r *http.Request, requestID string) (*UserContext, error) {
	return v.resolve(r, requestID)
}

func extractString(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
