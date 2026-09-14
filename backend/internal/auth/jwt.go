// package auth（与 password.go 同属 auth 包）。
// 本文件负责 JWT（JSON Web Token）的签发与解析，实现「无状态登录态」：
// 用户登录后拿到 token，之后每次请求在 Header 里带上，服务端验签即可知道是谁，无需查 session 表。
package auth

import (
	"fmt"  // 格式化错误，例如签名算法不符、token 无效
	"time" // 设置 token 签发时间 IssuedAt 和过期时间 ExpiresAt

	// github.com/golang-jwt/jwt/v5：社区常用的 JWT 库（v5 版 API）。
	// 负责把 Claims 结构体编码成 "header.payload.signature" 三段式字符串，以及反向解析验签。
	"github.com/golang-jwt/jwt/v5"
)

// Claims 是写入 JWT payload（载荷）里的自定义数据 + 标准字段。
// 解析 token 成功后，可从 claims.UserID 得到当前登录用户的数据库 id。
//
// 嵌入 jwt.RegisteredClaims：JWT 标准字段（签发时间、过期时间、issuer 等），
// 本项目的 SignToken 只设置了 IssuedAt 和 ExpiresAt。
//
// json:"user_id"：若将来把 Claims 序列化到 JSON，字段名为 user_id（JWT 库里也会按此编码进 payload）。
//
// 用在哪：
//   - SignToken 构造并签名
//   - ParseToken 解析后返回 *Claims
//   - middleware/auth.go ParseAuthUserID → claims.UserID 注入 request.Context
type Claims struct {
	UserID int64 `json:"user_id"` // 对应 users 表主键；中间件据此识别「当前是谁」
	jwt.RegisteredClaims          // 嵌入标准声明：ExpiresAt、IssuedAt 等（Go 嵌入字段语法）
}

// SignToken 为指定用户签发 JWT 字符串，登录/注册成功后返回给前端。
//
// 参数 secret：签名密钥，来自 config.JWTSecret（.env 的 JWT_SECRET），必须保密。
// 参数 userID：刚登录用户的 id，写入 Claims，后续 API 靠它查库。
// 参数 expiryHours：有效小时数，来自 config.JWTExpiryHours（默认 168=7 天）。
// 返回值：完整 token 字符串，前端通常存 localStorage，请求头带 Authorization: Bearer <token>。
//
// 用在哪：
//   - service/auth_service.go Register() / Login()：成功后 auth.SignToken(s.jwtSecret, userID, s.jwtExpiryHours)
//
// 算法：HS256（HMAC-SHA256），对称密钥，secret 同时用于签发和验签（适合单体后端）。
func SignToken(secret string, userID int64, expiryHours int) (string, error) {
	now := time.Now() // 当前时间，作为签发时刻
	// 填充 Claims：业务字段 + 标准过期信息
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			// IssuedAt（iat）：token 何时签发
			IssuedAt: jwt.NewNumericDate(now),
			// ExpiresAt（exp）：过期时刻 = 现在 + expiryHours 小时；过期后 ParseToken 会失败
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiryHours) * time.Hour)),
		},
	}
	// NewWithClaims：指定签名算法 HS256 + 上面的 claims，生成未签名的 token 对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// SignedString：用 secret 做 HMAC 签名，得到最终字符串，例如 "eyJhbGciOiJIUzI1NiIs..."
	return token.SignedString([]byte(secret))
}

// ParseToken 验证并解析客户端传来的 JWT，提取 Claims（主要是 UserID）。
//
// 参数 secret：必须与 SignToken 时相同，否则验签失败（防篡改）。
// 参数 raw：Authorization 头里 "Bearer " 后面的 token 字符串。
// 返回值1：解析成功时的 *Claims；失败时为 nil。
// 返回值2：过期、签名错误、格式错误等时非 nil。
//
// 用在哪：
//   - middleware/auth.go ParseAuthUserID()：
//     从请求头取 Bearer token → auth.ParseToken(jwtSecret, token) → return claims.UserID
//   - 所有 RequireAuth / RequireAdmin 保护的 API 都间接依赖此函数
//
// 安全：回调里强制要求签名方法必须是 HMAC，防止「alg:none」等已知 JWT 攻击。
func ParseToken(secret string, raw string) (*Claims, error) {
	// ParseWithClaims：解析 raw，并把 payload 解到 &Claims{}；第三个参数是「提供验签密钥」的回调。
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 检查 token 头里的 alg 是否为 HMAC 系列（本项目 SignToken 用的是 HS256）
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method") // 拒绝非 HMAC 算法
		}
		// 返回验签用的密钥（字节形式）；必须与 SignToken 的 secret 一致
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err // token 过期、签名不对、格式 malformed 等都会走这里
	}
	// token.Claims 是 interface{}，需断言为 *Claims；token.Valid 表示签名与过期时间都通过
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil // 成功：调用方可读 claims.UserID
}
