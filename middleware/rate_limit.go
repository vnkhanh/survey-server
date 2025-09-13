package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Mỗi IP có một limiter riêng + lastSeen để dọn dẹp
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter quản lý map<ip, limiter>
type IPRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor

	// cấu hình chung cho tất cả IP
	reqPerMin int // số request/phút
	burst     int // số burst cho phép
	ttl       time.Duration
}

// reqPerMin: ví dụ 10, burst: 5, ttl: 5 phút (IP không hoạt động sẽ bị dọn)
func NewIPRateLimiter(reqPerMin, burst int, ttl time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		visitors:  make(map[string]*visitor),
		reqPerMin: reqPerMin,
		burst:     burst,
		ttl:       ttl,
	}
	// chạy nền dọn IP cũ
	go rl.cleanupVisitors()
	return rl
}

func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if v, ok := rl.visitors[ip]; ok {
		v.lastSeen = time.Now()
		return v.limiter
	}

	// chuyển req/phút -> rate.Limit (req/giây)
	rps := float64(rl.reqPerMin) / 60.0
	limiter := rate.NewLimiter(rate.Limit(rps), rl.burst)
	rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

func (rl *IPRateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.ttl {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// ====== Middleware dùng cho 1 endpoint cụ thể ======

func RateLimitByIP(rl *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP() // Gin sẽ xét X-Forwarded-For nếu đã cấu hình TrustedProxies
		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "Too Many Requests",
				"hint":    "Vui lòng thử lại sau ít phút.",
			})
			return
		}
		c.Next()
	}
}

// ====== Một instance dùng riêng cho POST /api/forms ======

// 10 requests/phút/IP, burst 5, TTL giữ limiter tối đa 5 phút
var FormsCreateLimiter = NewIPRateLimiter(10, 5, 5*time.Minute)

// RateLimitFormsCreate: gắn vào route POST /api/forms
func RateLimitFormsCreate() gin.HandlerFunc {
	return RateLimitByIP(FormsCreateLimiter)
}
