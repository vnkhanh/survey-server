package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/middleware"
	"github.com/vnkhanh/survey-server/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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
		Where("trang_thai != ?", "archived")

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

// BE-luu: Lấy danh sách room đã lưu trữ
func GetArchivedRooms(c *gin.Context) {
	userVal, exists := c.Get(middleware.CtxUser)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Không tìm thấy user trong context"})
		return
	}
	u := userVal.(models.NguoiDung)

	// Lấy query param page & limit
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Query room archived của user + preload members
	var rooms []models.Room
	query := config.DB.Model(&models.Room{}).
		Where("trang_thai = ? AND nguoi_tao_id = ?", "archived", u.ID).
		Preload("Members")

	// Đếm tổng số room
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Không đếm được số room lưu trữ",
			"error":   err.Error(),
		})
		return
	}

	// Tính tổng số trang
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Lấy dữ liệu với phân trang
	if err := query.Offset(offset).Limit(limit).Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Không lấy được danh sách room lưu trữ",
			"error":   err.Error(),
		})
		return
	}

	// Trả về JSON
	c.JSON(http.StatusOK, gin.H{
		"data":       rooms,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
	})
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

// ShareRoom - BE-23: Tạo short link chia sẻ Room
func ShareRoom(c *gin.Context) {
	// Lấy id room từ param
	roomID := c.Param("id")

	var room models.Room
	if err := config.DB.First(&room, "id = ?", roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Room không tồn tại"})
		return
	}

	// ✅ Middleware CheckRoomOwner đã chạy trước nên ở đây room chắc chắn thuộc owner
	// Sinh UUID làm share_url
	shortLink := uuid.NewString()
	room.ShareURL = shortLink

	// Lưu vào DB
	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không tạo được share link"})
		return
	}

	// Trả kết quả
	c.JSON(http.StatusOK, gin.H{
		"message":   "Tạo share link thành công",
		"share_url": room.ShareURL,
		"data": gin.H{
			"id":         room.ID,
			"ten_room":   room.TenRoom,
			"mo_ta":      room.MoTa,
			"share_url":  room.ShareURL,
			"is_public":  room.IsPublic,
			"trang_thai": room.TrangThai,
		},
	})
}

// BE21 Lấy danh sách room public (lobby)
// BE: Lấy danh sách room trong lobby (phân trang + tìm kiếm theo tên)
func GetLobbyRooms(c *gin.Context) {
	var rooms []models.Room

	// Lấy query param page & limit
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Query cơ bản: chỉ lấy room active và chưa khóa
	query := config.DB.Model(&models.Room{}).
		Where("khoa = ? AND trang_thai = ?", false, "active")

	// Nếu có search thì thêm điều kiện tìm kiếm
	if search != "" {
		query = query.Where("ten_room ILIKE ?", "%"+search+"%")
	}

	// Đếm tổng số room
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không đếm được số room", "error": err.Error()})
		return
	}

	// Tính tổng số trang
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Lấy room (có preload members)
	if err := query.Preload("Members", "trang_thai = ?", "active").
		Offset(offset).Limit(limit).
		Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không lấy được danh sách room", "error": err.Error()})
		return
	}

	// Chuẩn bị response
	var result []gin.H
	for _, room := range rooms {
		result = append(result, gin.H{
			"id":           room.ID,
			"ten_room":     room.TenRoom,
			"mo_ta":        room.MoTa,
			"trang_thai":   room.TrangThai,
			"member_count": len(room.Members),
			"members":      room.Members,
			"is_public":    room.IsPublic,
		})
	}

	// Trả về JSON
	c.JSON(http.StatusOK, gin.H{
		"rooms":      result,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
	})
}

// BE22 Tham gia room (enter room)
func EnterRoom(c *gin.Context) {
	roomID := c.Param("id")

	// Tìm room
	var room models.Room
	if err := config.DB.First(&room, roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room không tồn tại"})
		return
	}

	// Lấy user từ context
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userVal.(models.NguoiDung)

	// Chỉ owner mới vào được nếu room bị khóa
	isOwner := room.NguoiTaoID != nil && *room.NguoiTaoID == user.ID
	if room.IsLocked && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Room đã bị khóa, không thể tham gia"})
		return
	}

	// Kiểm tra user đã là thành viên chưa
	var existing models.RoomNguoiThamGia
	err := config.DB.Where("room_id = ? AND nguoi_dung_id = ?", room.ID, user.ID).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// User chưa tham gia -> thêm mới
		participant := models.RoomNguoiThamGia{
			RoomID:       room.ID,
			NguoiDungID:  user.ID,
			TenNguoiDung: user.Ten,
			TrangThai:    "active",
			NgayVao:      time.Now(),
		}
		if err := config.DB.Create(&participant).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thêm được thành viên"})
			return
		}
	} else if err != nil {
		// Lỗi DB thật sự
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra thành viên"})
		return
	}
	// Nếu user đã tồn tại thì không thêm mới -> chỉ tiếp tục trả về danh sách

	// Lấy danh sách thành viên (active)
	var participants []struct {
		ID           uint   `json:"id"`
		UserID       uint   `json:"user_id"`
		TenNguoiDung string `json:"ten_nguoi_dung"`
		Status       string `json:"status"`
	}
	if err := config.DB.Model(&models.RoomNguoiThamGia{}).
		Select("id, nguoi_dung_id as user_id, ten_nguoi_dung, trang_thai as status").
		Where("room_id = ? AND trang_thai = ?", room.ID, "active").
		Scan(&participants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lấy được danh sách thành viên"})
		return
	}

	// Đếm số lượng thành viên unique
	var memberCount int64
	config.DB.Model(&models.RoomNguoiThamGia{}).
		Where("room_id = ? AND trang_thai = ?", room.ID, "active").
		Count(&memberCount)

	// Trả về
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"room": gin.H{
			"id":           room.ID,
			"ten_room":     room.TenRoom,
			"mo_ta":        room.MoTa,
			"share_url":    room.ShareURL,
			"is_public":    room.IsPublic,
			"ngay_tao":     room.NgayTao,
			"member_count": memberCount,
			"members":      participants,
		},
	})
}

// ===== API 22-2: Thêm thành viên vào room =====
// ✅ Gửi lời mời
func InviteUserToRoom(c *gin.Context) {
	roomID := c.Param("id")

	var body struct {
		UserID uint   `json:"user_id" binding:"required"`
		Email  string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Kiểm tra room tồn tại
	var room models.Room
	if err := config.DB.First(&room, roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room không tồn tại"})
		return
	}

	// Kiểm tra đã là thành viên chưa
	var existingMember models.RoomNguoiThamGia
	if err := config.DB.Where("room_id = ? AND nguoi_dung_id = ?", roomID, body.UserID).First(&existingMember).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người dùng đã là thành viên trong room"})
		return
	}

	// Kiểm tra đã có lời mời chưa
	var existingInvite models.RoomInvite
	if err := config.DB.Where("room_id = ? AND user_id = ?", roomID, body.UserID).First(&existingInvite).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đã gửi lời mời cho người dùng này"})
		return
	}

	// Tạo lời mời
	invite := models.RoomInvite{
		RoomID:    room.ID,
		UserID:    body.UserID,
		Email:     body.Email,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := config.DB.Create(&invite).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "Không thể gửi lời mời",
			"db_error": err.Error(), // in chi tiết lỗi DB
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Đã gửi lời mời",
		"invite":  invite,
	})
}

// ✅ 2. Xem danh sách lời mời trong room
func ListRoomInvites(c *gin.Context) {
	// Lấy user từ context
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userVal.(models.NguoiDung)

	roomID := c.Param("id")

	var invites []models.RoomInvite
	// Chỉ lấy lời mời trong room cho user hiện tại
	if err := config.DB.
		Where("room_id = ? AND user_id = ?", roomID, user.ID).
		Find(&invites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lấy được danh sách lời mời"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invites": invites})
}


// ✅ 3. Người dùng phản hồi lời mời (accept / reject)
func RespondToInvite(c *gin.Context) {
	inviteID := c.Param("inviteID")

	var body struct {
		Status string `json:"status" binding:"required,oneof=accepted rejected"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái không hợp lệ"})
		return
	}

	var invite models.RoomInvite
	if err := config.DB.First(&invite, inviteID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lời mời không tồn tại"})
		return
	}

	invite.Status = body.Status
	config.DB.Save(&invite)

	// Nếu user chấp nhận thì thêm vào RoomNguoiThamGia
	if body.Status == "accepted" {
		member := models.RoomNguoiThamGia{
			RoomID:       invite.RoomID,
			NguoiDungID:  invite.UserID,
			TenNguoiDung: invite.Email, // hoặc lấy từ bảng NguoiDung
			TrangThai:    "active",
			IP:           c.ClientIP(),
			NgayVao:      time.Now(),
		}
		config.DB.Create(&member)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Phản hồi lời mời thành công",
		"invite":  invite,
	})
}

// ✅ 4. Xóa lời mời
func DeleteInvite(c *gin.Context) {
	inviteID := c.Param("inviteID")

	if err := config.DB.Delete(&models.RoomInvite{}, inviteID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa lời mời"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Xóa lời mời thành công"})
}

// ===== API 22-3: Xóa thành viên khỏi room =====
func RemoveMemberFromRoom(c *gin.Context) {
	// Lấy roomID và memberID từ param
	roomID := c.Param("id")
	memberID := c.Param("memberId")

	roomIDUint, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID room không hợp lệ"})
		return
	}

	memberIDUint, err := strconv.ParseUint(memberID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID thành viên không hợp lệ"})
		return
	}

	// Lấy thông tin room
	var room models.Room
	if err := config.DB.First(&room, roomIDUint).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room không tồn tại"})
		return
	}

	// Thực hiện xóa dựa trên ID bản ghi RoomNguoiThamGia
	result := config.DB.Where("id = ? AND room_id = ?", memberIDUint, roomIDUint).
		Delete(&models.RoomNguoiThamGia{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không xóa được thành viên"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thành viên không tồn tại trong room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Xóa thành viên thành công",
	})
}

// ===== BE-29: Lấy danh sách thành viên trong room =====
func GetRoomParticipants(c *gin.Context) {
	roomID := c.Param("id")
	roomIDUint, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID room không hợp lệ"})
		return
	}

	// Lấy user từ context
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userVal.(models.NguoiDung)

	// Lấy room
	var room models.Room
	if err := config.DB.First(&room, roomIDUint).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room không tồn tại"})
		return
	}

	isPublic := true
	if room.IsPublic != nil {
		isPublic = *room.IsPublic
	}
	// Nếu room private, check quyền
	if !isPublic {
		var isParticipant int64
		config.DB.Model(&models.RoomNguoiThamGia{}).Where("room_id = ? AND nguoi_dung_id = ? AND trang_thai = ?", roomIDUint, user.ID, "active").Count(&isParticipant)
		if room.NguoiTaoID != nil && *room.NguoiTaoID != user.ID && isParticipant == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không có quyền xem danh sách thành viên"})
			return
		}
	}

	var participants []models.RoomNguoiThamGia
	if err := config.DB.Where("room_id = ? AND trang_thai = ?", roomIDUint, "active").Find(&participants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lấy được danh sách thành viên"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"room_id":      roomID,
		"participants": participants,
	})
}

// ===== BE-30: Khóa room =====
func LockRoom(c *gin.Context) {
	roomID := c.Param("id")
	roomIDUint, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID room không hợp lệ"})
		return
	}

	var room models.Room
	if err := config.DB.First(&room, roomIDUint).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room không tồn tại"})
		return
	}

	// ✅ Lấy user từ context
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userVal.(models.NguoiDung)

	// ✅ Check owner
	if room.NguoiTaoID == nil || *room.NguoiTaoID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ owner mới có quyền khóa room"})
		return
	}

	// ✅ Thực hiện khóa
	room.IsLocked = true
	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khóa room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Room đã bị khóa",
		"room_id":   room.ID,
		"is_locked": room.IsLocked,
	})
}

// ===== BE-31: Mở khóa room =====
// UnlockRoom mở khóa room (chỉ owner được phép)
func UnlockRoom(c *gin.Context) {
	roomID := c.Param("id")
	roomIDUint, err := strconv.ParseUint(roomID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID room không hợp lệ"})
		return
	}

	// Tìm room
	var room models.Room
	if err := config.DB.First(&room, roomIDUint).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room không tồn tại"})
		return
	}

	// Lấy user từ context
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userVal.(models.NguoiDung)

	// Chỉ owner được phép mở khóa
	if room.NguoiTaoID == nil || *room.NguoiTaoID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ owner mới được mở khóa room"})
		return
	}

	// Cập nhật trạng thái
	room.IsLocked = false
	if err := config.DB.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở khóa room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Room đã được mở khóa",
		"room_id":   room.ID,
		"is_locked": room.IsLocked,
	})
}
