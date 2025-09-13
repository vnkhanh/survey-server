package controllers

import (
	"net/http"
	"strconv"
	"strings"
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

// BE-13: danh sách room
func ListRooms(c *gin.Context) {
	var rooms []models.Room
	query := config.DB.Model(&models.Room{})

	// bỏ qua room đã xóa (delete)
	query = query.Where("trang_thai != ?", "delete")

	// filter theo owner_id
	if ownerID := c.Query("owner_id"); ownerID != "" {
		query = query.Where("nguoi_tao_id = ?", ownerID)
	}

	// filter theo từ khóa (tìm theo tên room)
	search := c.Query("search")
	if search != "" {
		query = query.Where(
			"LOWER(ten_room) LIKE ?",
			"%"+strings.ToLower(search)+"%",
		)
	}

	// phân trang
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int64
	query.Count(&total)

	// sắp xếp
	sortBy := c.DefaultQuery("sort_by", "created_at")                  // name | created_at
	sortOrder := strings.ToLower(c.DefaultQuery("sort_order", "desc")) // asc | desc

	orderClause := "ngay_tao desc"
	switch sortBy {
	case "name":
		if sortOrder == "asc" {
			orderClause = "ten_room asc"
		} else {
			orderClause = "ten_room desc"
		}
	case "created_at":
		if sortOrder == "asc" {
			orderClause = "ngay_tao asc"
		} else {
			orderClause = "ngay_tao desc"
		}
	}

	if err := query.Offset(offset).Limit(limit).Order(orderClause).Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không lấy được danh sách room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  rooms,
		"total": total,
		"page":  page,
		"limit": limit,
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
			"khao_sat": gin.H{
				"id":      form.ID,
				"tieu_de": form.TieuDe,
				"mo_ta":   form.MoTa,
			},
		},
	})
}

// BE-15: cập nhật room
func UpdateRoom(c *gin.Context) {
	// roomObj đã được middleware.CheckRoomOwner nạp vào context
	room := c.MustGet("roomObj").(models.Room)

	var req struct {
		TenRoom   *string `json:"ten_room"`
		MoTa      *string `json:"mo_ta"`
		KhaoSatID *uint   `json:"khao_sat_id"`
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
	if req.KhaoSatID != nil {
		// kiểm tra khảo sát tồn tại và hợp lệ
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

// BE-16: xoá room
func DeleteRoom(c *gin.Context) {
	// roomObj đã được middleware.CheckRoomOwner nạp vào context
	room := c.MustGet("roomObj").(models.Room)

	// Nếu đã inactive rồi thì báo luôn
	if room.TrangThai == "inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Room đã được lưu trữ trước đó"})
		return
	}

	// Đánh dấu inactive (archive)
	room.TrangThai = "inactive"

	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể lưu trữ room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Room đã được lưu trữ (archive)", "data": room})
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
	room.TrangThai = "locked"
	room.Khoa = true

	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không đặt được mật khẩu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đặt mật khẩu thành công", "data": room})
}
