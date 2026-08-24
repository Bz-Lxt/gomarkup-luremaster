package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e APIError) Error() string { return e.Code + ": " + e.Message }

var (
	ErrUnauthorized  = APIError{Code: "AUTH_UNAUTHORIZED", Message: "未登录或令牌失效", Status: http.StatusUnauthorized}
	ErrForbidden     = APIError{Code: "AUTH_FORBIDDEN", Message: "无权访问该资源", Status: http.StatusForbidden}
	ErrInvalidCreds  = APIError{Code: "AUTH_INVALID", Message: "邮箱或密码错误", Status: http.StatusUnauthorized}
	ErrConflictEmail = APIError{Code: "AUTH_CONFLICT", Message: "邮箱或用户名已存在", Status: http.StatusConflict}
	ErrValidation    = APIError{Code: "VALIDATION", Message: "请求参数不合法", Status: http.StatusBadRequest}
	ErrNotFound      = APIError{Code: "NOT_FOUND", Message: "资源不存在", Status: http.StatusNotFound}
	ErrSpotPrivate   = APIError{Code: "SPOT_PRIVATE", Message: "该标点为私有，坐标已脱敏", Status: http.StatusOK}
	ErrSlotTaken     = APIError{Code: "SLOT_TAKEN", Message: "席位已被占用", Status: http.StatusConflict}
	ErrSlotState     = APIError{Code: "SLOT_STATE", Message: "席位状态不允许该操作", Status: http.StatusConflict}
	ErrActivityState = APIError{Code: "ACTIVITY_STATE", Message: "活动状态不允许该操作", Status: http.StatusConflict}
	ErrCheckinFar    = APIError{Code: "CHECKIN_TOO_FAR", Message: "不在集合点半径内", Status: http.StatusBadRequest}
	ErrHydroPending  = APIError{Code: "HYDRO_PENDING", Message: "水文绑定尚未完成", Status: http.StatusAccepted}
	ErrRateLimited   = APIError{Code: "RATE_LIMITED", Message: "请求过于频繁", Status: http.StatusTooManyRequests}
	ErrInternal      = APIError{Code: "INTERNAL", Message: "服务内部错误", Status: http.StatusInternalServerError}
)

func Abort(c *gin.Context, err error) {
	var ae APIError
	if errors.As(err, &ae) {
		c.AbortWithStatusJSON(ae.Status, gin.H{"ok": false, "error": ae})
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"ok": false, "error": ErrInternal})
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"ok": true, "data": data})
}

func Validation(msg string) APIError {
	return APIError{Code: "VALIDATION", Message: msg, Status: http.StatusBadRequest}
}
