package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/requestlog"
	"personal_assistant_server/internal/response"
	"personal_assistant_server/internal/service"
)

// Auth 是认证相关 HTTP 接口的处理器，负责接收并校验客户端请求，
// 再将具体的认证、用户查询和验证码操作交由对应的业务服务完成。
//
// 处理器本身不保存登录状态：登录身份由客户端携带的令牌表示，
// 并由认证中间件解析后写入当前请求的 gin.Context。
type Auth struct {
	service        *service.Auth    // service 处理用户登录、令牌签发和用户信息查询。
	captchaService *service.Captcha // captchaService 负责验证码的生成、存储和校验。
}

// NewAuth 创建认证接口处理器。
// authService 和 captchaService 由程序启动时完成初始化并通过依赖注入传入，
// 使处理器只负责 HTTP 层的数据转换与错误响应，不直接依赖底层存储实现。
func NewAuth(authService *service.Auth, captchaService *service.Captcha) *Auth {
	return &Auth{service: authService, captchaService: captchaService}
}

// loginRequest 定义登录接口期望接收的 JSON 请求体。
// binding:"required" 用于要求 Gin 在绑定请求时校验所有字段均已提供且不为空。
type loginRequest struct {
	Username    string `json:"username" binding:"required"`    // Username 是用户提交的登录名。
	Password    string `json:"password" binding:"required"`    // Password 是用户提交的明文密码，仅用于本次认证校验。
	CaptchaID   string `json:"captchaId" binding:"required"`   // CaptchaID 用于定位此前由服务端生成的验证码。
	CaptchaCode string `json:"captchaCode" binding:"required"` // CaptchaCode 是用户根据验证码图片填写的答案。
}

// Captcha 生成一组新的登录验证码。
//
// 成功时返回验证码 ID、可供客户端展示的图片数据和 Unix 时间戳格式的过期时间。
// 客户端后续登录时必须同时提交 captchaId 和用户识别出的 captchaCode。
func (h *Auth) Captcha(c *gin.Context) {
	// 由验证码服务完成随机内容生成、图片编码以及有效期记录。
	id, image, expiresAt, err := h.captchaService.Create()
	if err != nil {
		// 生成失败属于服务端异常，避免将内部实现错误直接暴露给客户端。
		requestlog.Error(c, "验证码生成失败", err)
		response.Error(c, http.StatusInternalServerError, "验证码生成失败")
		return
	}

	// expiresAt 转换为秒级 Unix 时间戳，便于不同平台的客户端统一处理。
	response.OK(c, gin.H{"captchaId": id, "image": image, "expiresAt": expiresAt.Unix()})
}

// Login 校验登录请求，并在认证成功后签发访问令牌。
//
// 处理顺序为：绑定并校验 JSON 请求体、校验验证码、验证用户名和密码，
// 最后向客户端返回访问令牌、令牌过期时间以及当前用户信息。
func (h *Auth) Login(c *gin.Context) {
	var request loginRequest

	// 将 JSON 请求体绑定到 loginRequest；JSON 格式错误或任一必填字段为空时均会失败。
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "用户名、密码和验证码不能为空")
		return
	}

	// 在检查账号密码前先校验验证码，阻止无效或已过期验证码继续尝试登录。
	if !h.captchaService.Verify(request.CaptchaID, request.CaptchaCode) {
		response.Error(c, http.StatusBadRequest, "验证码错误或已过期")
		return
	}

	// 认证服务负责核对用户凭据，并在成功后生成具有有效期的访问令牌。
	user, token, expiresAt, err := h.service.Login(request.Username, request.Password)

	// 无效凭据是可预期的认证失败，返回 401；不向客户端区分用户名或密码错误，
	// 以免攻击者据此判断某个用户名是否存在。
	if errors.Is(err, service.ErrInvalidCredentials) {
		response.Error(c, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 其他错误视为服务端内部故障，不向客户端泄露具体错误细节。
	if err != nil {
		requestlog.Error(c, "登录失败", err, "username", request.Username)
		response.Error(c, http.StatusInternalServerError, "登录失败")
		return
	}

	// 令牌过期时间使用秒级 Unix 时间戳，user 中包含客户端展示所需的用户资料。
	response.OK(c, gin.H{"token": token, "expiresAt": expiresAt.Unix(), "user": user})
}

// Profile 返回当前已登录用户的资料。
// 该接口依赖认证中间件预先解析访问令牌，并将令牌声明写入请求上下文。
func (h *Auth) Profile(c *gin.Context) {
	// 从当前请求上下文中读取认证中间件写入的令牌声明。
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		// 上下文中没有有效声明，说明请求尚未通过身份认证。
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return
	}

	// 使用令牌中的用户 ID 查询最新资料，避免直接返回令牌内可能过时的数据。
	user, err := h.service.UserByID(claims.UserID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "用户不存在")
		return
	}

	response.OK(c, user)
}
