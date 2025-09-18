package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/vnkhanh/survey-server/middleware"
	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
)

// BE-12: tạo room
func CreateRoom(c *gin.Context) {
	u := c.MustGet(middleware.CtxUser).(models.NguoiDung)

	var req struct {
		KhaoSatID uint    `json:"khao_sat_id" binding:"required"`
		TenRoom   string  `json:"ten_room" binding:"required"`
		MoTa      *string `json:"mo_ta"`
		IsPublic  *bool   `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Dữ liệu không hợp lệ",
			"error":   err.Error(),
		})
		return
	}

	// kiểm tra trạng thái khảo sát
	var ks models.KhaoSat
	if err := config.DB.First(&ks, req.KhaoSatID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Khảo sát không tồn tại"})
		return
	}

	if ks.TrangThai != "published" && ks.TrangThai != "active" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Chỉ có thể tạo room từ khảo sát đang hoạt động",
		})
		return
	}

	// tạo share url
	shareURL := uuid.NewString()

	room := models.Room{
		KhaoSatID:  req.KhaoSatID,
		TenRoom:    req.TenRoom,
		MoTa:       req.MoTa,
		NguoiTaoID: &u.ID,
		TrangThai:  "active",
		Khoa:       false,
		IsPublic:   req.IsPublic,
		NgayTao:    time.Now(),
		ShareURL:   shareURL,
	}

	if err := config.DB.Create(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không tạo được room"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tạo room thành công",
		"data":    room,
	})
}

// BE-13: danh sách room của người quản lý
func ListRooms(c *gin.Context) {
	u := c.MustGet(middleware.CtxUser).(models.NguoiDung) // lấy user từ token

	var rooms []models.Room
	query := config.DB.Model(&models.Room{}).
		Where("nguoi_tao_id = ?", u.ID). // chỉ lấy room do user này tạo
		Where("trang_thai != ?", "deleted")

	// filter theo tên (q)
	if q := c.Query("q"); q != "" {
		query = query.Where("ten_room LIKE ?", "%"+q+"%")
	}

	// phân trang
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int64
	query.Count(&total)

	if err := query.
		Limit(limit).
		Offset(offset).
		Order("ngay_tao DESC").
		Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể lấy danh sách room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  rooms,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// BE-14: lấy chi tiết room
func GetRoomDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID không hợp lệ"})
		return
	}

	var room models.Room
	if err := config.DB.First(&room, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Room không tồn tại"})
		return
	}

	// Lấy form liên kết
	var form models.KhaoSat
	config.DB.First(&form, room.KhaoSatID)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":         room.ID,
			"ten_room":   room.TenRoom,
			"mo_ta":      room.MoTa,
			"trang_thai": room.TrangThai,
			"khoa":       room.Khoa,
			"share_url":  room.ShareURL,
			"is_public":  room.IsPublic,
			"khao_sat": gin.H{
				"id":      form.ID,
				"tieu_de": form.TieuDe,
				"mo_ta":   form.MoTa,
			},
		},
	})
}

func UpdateRoom(c *gin.Context) {
	// roomObj đã được middleware.CheckRoomOwner nạp vào context
	room := c.MustGet("roomObj").(models.Room)

	var req struct {
		TenRoom   *string `json:"ten_room"`
		MoTa      *string `json:"mo_ta"`
		KhaoSatID *uint   `json:"khao_sat_id"`
		IsPublic  *bool   `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dữ liệu không hợp lệ"})
		return
	}

	// update từng field nếu có
	if req.TenRoom != nil {
		room.TenRoom = *req.TenRoom
	}
	if req.MoTa != nil {
		room.MoTa = req.MoTa
	}
	if req.IsPublic != nil {
		room.IsPublic = req.IsPublic
	}
	if req.KhaoSatID != nil {
		var ks models.KhaoSat
		if err := config.DB.First(&ks, *req.KhaoSatID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "Khảo sát không tồn tại"})
			return
		}
		if ks.TrangThai != "published" && ks.TrangThai != "active" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Chỉ có thể liên kết room với khảo sát đã publish hoặc đang active"})
			return
		}
		room.KhaoSatID = *req.KhaoSatID
	}

	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không cập nhật được room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật room thành công",
		"data":    room,
	})
}

// BE-16: xoá room (hard delete - xóa hẳn khỏi DB)
func DeleteRoom(c *gin.Context) {
	// roomObj đã được middleware.CheckRoomOwner nạp vào context
	room := c.MustGet("roomObj").(models.Room)

	// Xóa trực tiếp trong database
	if err := config.DB.Delete(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể xóa room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Room đã được xóa vĩnh viễn"})
}

// BE-16: lưu trữ room
func ArchiveRoom(c *gin.Context) {
	// roomObj đã được middleware.CheckRoomOwner nạp vào context
	room := c.MustGet("roomObj").(models.Room)

	// Nếu đã archived rồi thì báo luôn
	if room.TrangThai == "archived" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Room đã được lưu trữ rồi"})
		return
	}

	// Đánh dấu archived (archived)
	room.TrangThai = "archived"
	falseVal := false
	room.IsPublic = &falseVal

	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Đã lưu trữ room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Room đã được lưu trữ", "data": room})
}

// BE-16: khôi phục room
func RestoreRoom(c *gin.Context) {
	// roomObj đã được middleware.CheckRoomOwner nạp vào context
	room := c.MustGet("roomObj").(models.Room)

	// Nếu đã restored rồi thì báo luôn
	if room.TrangThai == "active" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Room đã được khôi phục rồi"})
		return
	}

	// Đánh dấu restored (restored)
	room.TrangThai = "active"

	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Đã khôi phục room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Room đã được khôi phục", "data": room})
}

// BE-17: đặt khoá room
func SetRoomPassword(c *gin.Context) {
	// roomObj đã được middleware.CheckRoomOwner nạp vào context
	room := c.MustGet("roomObj").(models.Room)

	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Password không hợp lệ"})
		return
	}

	// Hash mật khẩu
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không hash được mật khẩu"})
		return
	}
	pwd := string(hash)

	// Cập nhật room
	room.MatKhau = &pwd
	room.Khoa = true

	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không đặt được mật khẩu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đặt mật khẩu thành công", "data": room})
}
