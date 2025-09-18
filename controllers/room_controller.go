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

	if req.TenRoom != nil {
		room.TenRoom = *req.TenRoom
	}
	if req.MoTa != nil {
		room.MoTa = req.MoTa
	}
	if req.KhaoSatID != nil {
		room.KhaoSatID = *req.KhaoSatID
	}

	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không cập nhật được room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật room thành công", "data": room})
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

// BE-18: gỡ mật khẩu room
func RemoveRoomPassword(c *gin.Context) {
	room := c.MustGet("roomObj").(models.Room)

	room.MatKhau = nil
	room.Khoa = false
	if room.TrangThai == "locked" {
		room.TrangThai = "active"
	}
	config.DB.Save(&room)
	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không gỡ được mật khẩu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã gỡ mật khẩu Room", "data": room})
}

// Tạo short link / token chia sẻ room
func CreateRoomShare(c *gin.Context) {
	// roomObj đã được middleware.CheckRoomOwner nạp vào context
	room := c.MustGet("roomObj").(models.Room)

	// Kiểm tra quyền chia sẻ theo setting
	if room.Khoa {
		c.JSON(http.StatusForbidden, gin.H{"message": "Room đang bị khoá, không thể chia sẻ"})
		return
	}

	// Tạo short link hoặc token
	shortLink := uuid.NewString()
	room.ShareURL = shortLink

	// Lưu DB
	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không tạo được share link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"share_url": room.ShareURL,
	})
}

// BE21 Lấy danh sách room public (lobby)
func GetLobbyRooms(c *gin.Context) {
	var rooms []models.Room

	if err := config.DB.
		Where("khoa = ? AND trang_thai = ?", false, "active").
		Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không lấy được danh sách room", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rooms})
}

// BE22 Tham gia room (enter room)
func EnterRoom(c *gin.Context) {
	// Lấy user hiện tại
	u := c.MustGet(middleware.CtxUser).(models.NguoiDung)

	// Lấy room ID từ path
	id := c.Param("id")

	var room models.Room
	if err := config.DB.First(&room, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Room không tồn tại"})
		return
	}

	// Nếu room có khóa
	if room.Khoa && room.MatKhau != nil {
		var req struct {
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Room yêu cầu mật khẩu"})
			return
		}

		// So sánh mật khẩu
		if err := bcrypt.CompareHashAndPassword([]byte(*room.MatKhau), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Mật khẩu không đúng"})
			return
		}
	}

	// Tạo participant record
	participant := models.RoomNguoiThamGia{
		RoomID:      room.ID,
		NguoiDungID: u.ID,
		TrangThai:   "active",
	}

	if err := config.DB.Create(&participant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể tham gia room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"participant_id": participant.ID,
	})
}
