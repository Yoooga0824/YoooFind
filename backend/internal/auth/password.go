// package auth 声明本文件属于 auth 包（认证相关工具）。
// 本包不直接处理 HTTP 请求，只提供「密码哈希」和「JWT 签发/解析」等底层能力，
// 由 service 层（AuthService、AdminService）和 middleware 层调用。
package auth

// 引入 bcrypt：Go 官方扩展库 golang.org/x/crypto/bcrypt，专门做密码单向哈希。
// 特点：同一明文每次哈希结果不同（内置随机盐），且计算较慢，暴力破解成本高。
// 数据库里只存哈希，绝不存明文密码。
import "golang.org/x/crypto/bcrypt"

// HashPassword 把用户输入的明文密码转成 bcrypt 哈希字符串，用于写入数据库。
//
// 参数 raw：明文密码，例如用户注册时填的 "abc123"。
// 返回值1：哈希字符串（以 "$2a$" 开头），可安全存入 users 表的 password_hash 字段。
// 返回值2：bcrypt 内部出错时非 nil（极少见，如 cost 参数非法）。
//
// 用在哪：
//   - service/auth_service.go Register()：注册时对密码哈希后 Create 用户
//   - service/admin_service.go UpdateUserPassword()：管理员重置用户密码时重新哈希
//
// 用法示例（在 Service 里）：
//   hash, err := auth.HashPassword(password)
//   if err != nil { return ..., fmt.Errorf("密码处理失败") }
//   s.users.Create(ctx, email, hash)
func HashPassword(raw string) (string, error) {
	// GenerateFromPassword：bcrypt 核心函数。
	// []byte(raw)：Go 字符串转字节切片，bcrypt 只接受 []byte。
	// bcrypt.DefaultCost：工作因子，默认 10；数值越大越慢、越安全，一般 10~12 即可。
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err // 向上层返回空字符串 + 错误，由 Service 转成友好提示
	}
	// hashed 是 []byte，数据库/JSON 通常存 string，这里显式转换。
	return string(hashed), nil
}

// VerifyPassword 校验「用户这次输入的明文」是否与「数据库里存的哈希」匹配。
//
// 参数 raw：用户登录时输入的明文密码。
// 参数 hashed：从数据库读出的 password_hash（HashPassword 曾经生成的那串）。
// 返回值 bool：true=密码正确，false=错误或哈希格式无效。
//
// 注意：CompareHashAndPassword 不会返回具体错误给调用方，这里统一封装成 bool，
// 避免向客户端泄露「用户是否存在」以外的信息（登录失败统一说「账号或密码错误」）。
//
// 用在哪：
//   - service/auth_service.go Login()：查用户后 auth.VerifyPassword(password, user.PasswordHash)
//
// 用法示例：
//   if !auth.VerifyPassword(password, user.PasswordHash) {
//       return ..., fmt.Errorf("账号或密码错误")
//   }
func VerifyPassword(raw string, hashed string) bool {
	// CompareHashAndPassword：从 hashed 里解析盐，再哈希 raw，与 stored hash 比对。
	// 参数顺序：先哈希、后明文（与 HashPassword 的输入顺序相反，注意别写反）。
	// 返回 nil 表示匹配；非 nil 表示不匹配或 hashed 格式损坏。
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(raw)) == nil
}
