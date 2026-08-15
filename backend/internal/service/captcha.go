package service

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"
	"sync"
	"time"
)

const (
	captchaLength = 4
	captchaTTL    = 2 * time.Minute
	captchaChars  = "23456789"

	// 验证码图片渲染参数。调整这些值可以统一控制图片尺寸、干扰程度和字符布局。
	captchaImageWidth        = 140 // 图片宽度，单位为像素。
	captchaImageHeight       = 48  // 图片高度，单位为像素。
	captchaNoiseBytes        = 256 // 每张图片预先生成的随机字节数，供干扰线、噪点和字符样式使用。
	captchaInterferenceLines = 9   // 穿过图片的随机干扰线数量，越多越难识别。
	captchaNoisePoints       = 80  // 随机噪点数量，越多越难识别，但过多会影响可读性。
	captchaHorizontalPadding = 6   // 字符分布区域与图片左右边缘之间的最小留白，单位为像素。
	captchaDigitStartY       = 10  // 字符相对于图片上边缘的基础纵坐标，单位为像素。
	captchaDigitScale        = 3   // 点阵字符的像素缩放倍数，数值越大字符越大。
	captchaDigitJitterX      = 5   // 字符横坐标的随机抖动范围，实际偏移约为正负该值的一半。
	captchaDigitJitterY      = 6   // 字符纵坐标的随机偏移范围：[0, captchaDigitJitterY)，单位为像素。
	captchaGlyphWidth        = 5   // 点阵字形的固定列数，用于计算字符居中位置。
)

type captchaEntry struct {
	answerHash [sha256.Size]byte
	expiresAt  time.Time
}

// Captcha 管理登录验证码的生成、过期和一次性校验。
//
// 验证码仅保存在当前服务进程内，并且只保存规范化答案的 SHA-256 摘要。所有读写均由
// 互斥锁保护，可以安全地被多个 HTTP 请求并发调用。
type Captcha struct {
	mu      sync.Mutex
	entries map[string]captchaEntry
	now     func() time.Time
}

// NewCaptcha 创建验证码服务。
//
// 返回值：
//   - *Captcha：使用内存存储验证码、默认有效期为两分钟的验证码服务。
func NewCaptcha() *Captcha {
	return &Captcha{entries: make(map[string]captchaEntry), now: time.Now}
}

// Create 生成一个新的登录验证码。
//
// 返回值：
//   - string：不可预测的验证码 ID，用于登录时关联验证码答案。
//   - string：可直接赋给 img src 的 PNG Data URL。
//   - time.Time：验证码失效时间。
//   - error：安全随机数生成失败时返回的错误。
func (c *Captcha) Create() (string, string, time.Time, error) {
	idBytes, err := randomBytes(18)
	if err != nil {
		return "", "", time.Time{}, err
	}
	codeBytes, err := randomBytes(captchaLength)
	if err != nil {
		return "", "", time.Time{}, err
	}
	noise, err := randomBytes(captchaNoiseBytes)
	if err != nil {
		return "", "", time.Time{}, err
	}

	code := make([]byte, captchaLength)
	for index := range code {
		code[index] = captchaChars[int(codeBytes[index])%len(captchaChars)]
	}

	id := base64.RawURLEncoding.EncodeToString(idBytes)
	expiresAt := c.now().Add(captchaTTL)
	entry := captchaEntry{answerHash: hashCaptchaAnswer(string(code)), expiresAt: expiresAt}

	imageData, err := renderCaptchaPNG(string(code), noise)
	if err != nil {
		return "", "", time.Time{}, err
	}

	c.mu.Lock()
	for existingID, existing := range c.entries {
		if !existing.expiresAt.After(c.now()) {
			delete(c.entries, existingID)
		}
	}
	c.entries[id] = entry
	c.mu.Unlock()

	return id, imageData, expiresAt, nil
}

// Verify 校验并销毁一个验证码。
//
// 无论答案是否正确，只要验证码 ID 存在，本次校验后都会将其销毁，以限制重复猜测。
// 答案比较不区分字母大小写。
//
// 参数：
//   - id：创建验证码时返回的验证码 ID。
//   - answer：用户输入的验证码内容。
//
// 返回值：
//   - bool：验证码存在、未过期且答案一致时返回 true，否则返回 false。
func (c *Captcha) Verify(id, answer string) bool {
	c.mu.Lock()
	entry, exists := c.entries[id]
	if exists {
		delete(c.entries, id)
	}
	c.mu.Unlock()

	if !exists || !entry.expiresAt.After(c.now()) {
		return false
	}
	actualHash := hashCaptchaAnswer(answer)
	return subtle.ConstantTimeCompare(entry.answerHash[:], actualHash[:]) == 1
}

// randomBytes 使用加密安全的随机源生成指定长度的随机字节。
//
// 参数：
//   - length：需要生成的字节数。
//
// 返回值：
//   - []byte：生成的随机字节。
//   - error：读取系统安全随机源失败时返回的错误。
func randomBytes(length int) ([]byte, error) {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		return nil, fmt.Errorf("generate captcha randomness: %w", err)
	}
	return value, nil
}

// hashCaptchaAnswer 规范化验证码答案并计算其 SHA-256 摘要。
//
// 规范化过程会移除答案首尾的空白字符，并将字母统一转换为大写。
func hashCaptchaAnswer(answer string) [sha256.Size]byte {
	return sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(answer))))
}

// renderCaptchaPNG 将验证码内容渲染为包含干扰线和噪点的 PNG Data URL。
//
// 参数：
//   - code：需要绘制的验证码内容，其中每个字符都必须存在对应的点阵字形。
//   - noise：用于确定干扰线、字符偏移、字符颜色和噪点位置的随机字节。
//
// 返回值：
//   - string：可直接赋给 img src 的 PNG Data URL。
//   - error：验证码包含不支持的字符或 PNG 编码失败时返回的错误。
func renderCaptchaPNG(code string, noise []byte) (string, error) {
	for index := range code {
		if _, exists := captchaGlyphPatterns[code[index]]; !exists {
			return "", fmt.Errorf("render captcha: unsupported character %q", code[index])
		}
	}

	canvas := image.NewRGBA(image.Rect(0, 0, captchaImageWidth, captchaImageHeight))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.RGBA{R: 238, G: 246, B: 249, A: 255}}, image.Point{}, draw.Src)

	for index := 0; index < captchaInterferenceLines; index++ {
		offset := index * 4
		lineColor := color.RGBA{
			R: 90 + noise[offset]%70,
			G: 120 + noise[offset+1]%70,
			B: 140 + noise[offset+2]%70,
			A: 105,
		}
		drawCaptchaLine(canvas,
			int(noise[offset])%captchaImageWidth, int(noise[offset+1])%captchaImageHeight,
			int(noise[offset+2])%captchaImageWidth, int(noise[offset+3])%captchaImageHeight,
			lineColor,
		)
	}

	for index, character := range code {
		x := captchaGlyphX(index, len(code), noise[32+index])
		y := captchaDigitStartY + int(noise[40+index]%captchaDigitJitterY)
		textColor := color.RGBA{R: 18 + noise[48+index]%30, G: 65 + noise[56+index]%35, B: 90 + noise[64+index]%35, A: 255}
		drawCaptchaGlyph(canvas, byte(character), x, y, captchaDigitScale, textColor)
	}

	for index := 0; index < captchaNoisePoints; index++ {
		offset := 96 + index*2
		x, y := int(noise[offset])%captchaImageWidth, int(noise[offset+1])%captchaImageHeight
		canvas.Set(x, y, color.RGBA{R: 54, G: 135, B: 170, A: 150})
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, canvas); err != nil {
		return "", fmt.Errorf("encode captcha image: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

// captchaGlyphX 将可用宽度按字符数量划分为等宽槽位，并返回字符在对应槽位内的居中横坐标。
func captchaGlyphX(index, characterCount int, randomByte byte) int {
	if characterCount <= 0 {
		return captchaHorizontalPadding
	}
	availableWidth := captchaImageWidth - 2*captchaHorizontalPadding
	glyphWidth := captchaGlyphWidth * captchaDigitScale
	slotCenter := captchaHorizontalPadding + (2*index+1)*availableWidth/(2*characterCount)
	jitter := int(randomByte%captchaDigitJitterX) - captchaDigitJitterX/2
	return slotCenter - glyphWidth/2 + jitter
}

// captchaGlyphPatterns 定义验证码支持字符的 5×7 点阵字形。
var captchaGlyphPatterns = map[byte][7]string{
	'2': {"11111", "00001", "00001", "11111", "10000", "10000", "11111"},
	'3': {"11111", "00001", "00001", "11111", "00001", "00001", "11111"},
	'4': {"10001", "10001", "10001", "11111", "00001", "00001", "00001"},
	'5': {"11111", "10000", "10000", "11111", "00001", "00001", "11111"},
	'6': {"11111", "10000", "10000", "11111", "10001", "10001", "11111"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"11111", "10001", "10001", "11111", "10001", "10001", "11111"},
	'9': {"11111", "10001", "10001", "11111", "00001", "00001", "11111"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01111", "10000", "10000", "10111", "10001", "10001", "01111"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'J': {"00111", "00010", "00010", "00010", "10010", "10010", "01100"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
}

// drawCaptchaGlyph 按指定缩放比例和颜色将一个点阵字符绘制到画布上。
func drawCaptchaGlyph(canvas *image.RGBA, character byte, startX, startY, scale int, textColor color.RGBA) {
	pattern := captchaGlyphPatterns[character]
	for row, pixels := range pattern {
		for column, pixel := range pixels {
			if pixel != '1' {
				continue
			}
			for offsetY := 0; offsetY < scale; offsetY++ {
				for offsetX := 0; offsetX < scale; offsetX++ {
					canvas.Set(startX+column*scale+offsetX, startY+row*scale+offsetY, textColor)
				}
			}
		}
	}
}

// drawCaptchaLine 使用 Bresenham 直线算法在两个坐标之间绘制干扰线。
func drawCaptchaLine(canvas *image.RGBA, x0, y0, x1, y1 int, lineColor color.RGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	stepX, stepY := -1, -1
	if x0 < x1 {
		stepX = 1
	}
	if y0 < y1 {
		stepY = 1
	}
	err := dx + dy
	for {
		canvas.Set(x0, y0, lineColor)
		if x0 == x1 && y0 == y1 {
			return
		}
		doubleError := 2 * err
		if doubleError >= dy {
			err += dy
			x0 += stepX
		}
		if doubleError <= dx {
			err += dx
			y0 += stepY
		}
	}
}

// abs 返回整数的绝对值。
func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
