package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/middleware"
	"github.com/vnkhanh/survey-server/models"
	"github.com/vnkhanh/survey-server/utils"
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
	u := c.MustGet(middleware.CtxUser).(models.NguoiDung)

	// Base query
	baseQuery := config.DB.Table("room").
		Joins("LEFT JOIN room_nguoi_tham_gia rntg ON rntg.room_id = room.id").
		Where("(room.nguoi_tao_id = ? OR rntg.nguoi_dung_id = ?)", u.ID, u.ID).
		Where("room.trang_thai != ?", "archived").
		Group("room.id")

	// Filter theo tên
	if q := c.Query("q"); q != "" {
		baseQuery = baseQuery.Where("room.ten_room LIKE ?", "%"+q+"%")
	}

	// Phân trang
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	// ---- Đếm tổng số ----
	var total int64
	if err := config.DB.Table("room").
		Joins("LEFT JOIN room_nguoi_tham_gia rntg ON rntg.room_id = room.id").
		Where("(room.nguoi_tao_id = ? OR rntg.nguoi_dung_id = ?)", u.ID, u.ID).
		Where("room.trang_thai != ?", "archived").
		Group("room.id").
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể đếm rooms"})
		return
	}

	// ---- Lấy dữ liệu ----
	var rooms []models.Room
	if err := baseQuery.
		Select("room.*").
		Limit(limit).
		Offset(offset).
		Order("room.ngay_tao DESC").
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
	param := c.Param("id")
	var room models.Room
	var err error

	// Kiểm tra xem param có phải số không
	if id, convErr := strconv.Atoi(param); convErr == nil && id > 0 {
		// Nếu là số => tìm theo ID
		err = config.DB.
			Preload("KhaoSat").
			Preload("Members.NguoiDung").
			Preload("NguoiTao"). // preload người tạo
			First(&room, id).Error
	} else {
		// Nếu không phải số => tìm theo ShareURL
		err = config.DB.
			Preload("KhaoSat").
			Preload("Members.NguoiDung").
			Preload("NguoiTao"). // preload người tạo
			Where("share_url = ?", param).
			First(&room).Error
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phòng"})
		return
	}

	var form models.KhaoSat
	if err := config.DB.First(&form, room.KhaoSatID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Khảo sát không tồn tại"})
		return
	}

	// Lấy public_link ưu tiên từ DB, nếu null thì fallback sang local URL
	var publicLink string
	if form.PublicLink != nil && *form.PublicLink != "" {
		publicLink = *form.PublicLink
	} else {
		publicLink = fmt.Sprintf("%s/survey/%d", c.Request.Host, form.ID)
		if !strings.HasPrefix(publicLink, "http") {
			publicLink = "http://" + publicLink
		}
	}

	// Build danh sách thành viên (không bao gồm owner)
	members := make([]gin.H, 0)
	ownerID := uint(0)
	if room.NguoiTaoID != nil {
		ownerID = *room.NguoiTaoID
	}

	for _, m := range room.Members {
		// Bỏ qua owner trong danh sách members vì sẽ hiển thị riêng
		if m.NguoiDungID == ownerID {
			continue
		}
		
		memberName := m.TenNguoiDung
		if memberName == "" && m.NguoiDung.Ten != "" {
			memberName = m.NguoiDung.Ten
		}
		if memberName == "" {
			memberName = "Người dùng"
		}

		members = append(members, gin.H{
			"id":            m.ID,
			"user_id":       m.NguoiDungID, // thêm user_id để FE dễ xử lý
			"nguoi_dung_id": m.NguoiDungID,
			"ten":           memberName,
			"name":          memberName, // thêm field name cho FE
			"email":         m.NguoiDung.Email,
			"trang_thai":    m.TrangThai,
			"status":        m.TrangThai, // thêm field status cho FE
			"ngay_vao":      m.NgayVao,
			"ip":            m.IP,
		})
	}

	// Tạo response với nguoi_tao đầy đủ thông tin
	nguoiTaoResponse := gin.H{
		"id":    uint(0),
		"ten":   "",
		"name":  "",
		"email": "",
	}

	if room.NguoiTaoID != nil && room.NguoiTao.ID > 0 {
		nguoiTaoResponse = gin.H{
			"id":    room.NguoiTao.ID,
			"ten":   room.NguoiTao.Ten,
			"name":  room.NguoiTao.Ten,
			"email": room.NguoiTao.Email,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":            room.ID,
			"ten_room":      room.TenRoom,
			"mo_ta":         room.MoTa,
			"trang_thai":    room.TrangThai,
			"khoa":          room.Khoa,
			"share_url":     room.ShareURL,
			"is_public":     room.IsPublic,
			"is_locked":     room.IsLocked,
			"nguoi_tao_id":  room.NguoiTaoID,
			"nguoi_tao_ten": room.NguoiTao.Ten,   // thêm tên owner riêng
			"nguoi_tao_email": room.NguoiTao.Email, // thêm email owner riêng
			"nguoi_tao":     nguoiTaoResponse,
			"khao_sat": gin.H{
				"id":          form.ID,
				"tieu_de":     form.TieuDe,
				"mo_ta":       form.MoTa,
				"public_link": publicLink,
			},
			"members":      members,
			"member_count": len(members) + 1, // +1 cho owner
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
	room.Khoa = false

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

// ShareRoom - BE-20: Tạo short link chia sẻ Room
func ShareRoom(c *gin.Context) {
	roomID := c.Param("id")

	userVal, exists := c.Get(middleware.CtxUser)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Bạn cần đăng nhập"})
		return
	}
	user := userVal.(models.NguoiDung)

	var room models.Room
	if err := config.DB.First(&room, "id = ?", roomID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Room không tồn tại"})
		return
	}

	// check user có trong phòng chưa
	var participant models.RoomNguoiThamGia
	if err := config.DB.Where("room_id = ? AND nguoi_dung_id = ?", room.ID, user.ID).
		First(&participant).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"message": "Bạn chưa tham gia phòng, không thể tạo link share"})
		return
	}

	// tạo shareURL nếu chưa có
	if room.ShareURL == "" {
		room.ShareURL = uuid.NewString()
		if err := config.DB.Save(&room).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Không tạo được share link"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Tạo share link thành công",
		"share_url": "/api/rooms/share/" + room.ShareURL,
	})
}

// BE22.1: Lấy thông tin room qua shareURL (public, không cần login)
func GetRoomByShareURL(c *gin.Context) {
	token := c.Param("shareURL")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Thiếu token share"})
		return
	}

	var room models.Room
	if err := config.DB.Where("share_url = ?", token).First(&room).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Room không tồn tại"})
		return
	}

	response := gin.H{
		"id":               room.ID,
		"ten_room":         room.TenRoom,
		"mo_ta":            room.MoTa,
		"is_public":        room.IsPublic,
		"locked":           room.Khoa,
		"require_password": room.MatKhau != nil && *room.MatKhau != "",
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Lấy thông tin room thành công",
		"room":    response,
	})
}

// EnterRoomByShareURL: tham gia room qua link share
func EnterRoomByShareURL(c *gin.Context) {
	shareURL := c.Param("shareURL")
	if shareURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu shareURL"})
		return
	}

	var room models.Room
	if err := config.DB.Where("share_url = ?", shareURL).First(&room).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room không tồn tại"})
		return
	}

	// Lấy user từ context
	userVal, exists := c.Get(middleware.CtxUser)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userVal.(models.NguoiDung)

	// Kiểm tra room khóa
	isOwner := room.NguoiTaoID != nil && *room.NguoiTaoID == user.ID
	if room.Khoa && !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Room đã bị khóa, không thể tham gia"})
		return
	}
	// Kiểm tra mật khẩu nếu có
	if room.MatKhau != nil && *room.MatKhau != "" {
		var body struct {
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập mật khẩu"})
			return
		}
		if !utils.CheckPassword(*room.MatKhau, body.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Sai mật khẩu"})
			return
		}
	}

	// Kiểm tra user đã là thành viên chưa
	var participant models.RoomNguoiThamGia
	err := config.DB.Where("room_id = ? AND nguoi_dung_id = ?", room.ID, user.ID).First(&participant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		participant = models.RoomNguoiThamGia{
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra thành viên"})
		return
	} else if participant.TrangThai != "active" {
		participant.TrangThai = "active"
		config.DB.Save(&participant)
	}

	// Lấy danh sách members active
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

	// Đếm số thành viên
	var memberCount int64
	config.DB.Model(&models.RoomNguoiThamGia{}).
		Where("room_id = ? AND trang_thai = ?", room.ID, "active").
		Count(&memberCount)

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

	// Lấy room (có preload members và NguoiDung)
	if err := query.Preload("Members.NguoiDung").
		Offset(offset).Limit(limit).
		Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không lấy được danh sách room", "error": err.Error()})
		return
	}

	var result []gin.H
	for _, room := range rooms {
		var members []gin.H

		// Map RoomNguoiThamGia -> JSON
		for _, m := range room.Members {
			if m.NguoiDung.ID == 0 {
				continue
			}
			member := gin.H{
				"id":     m.NguoiDung.ID,
				"name":   m.NguoiDung.Ten,
				"email":  m.NguoiDung.Email,
				"status": m.TrangThai, // sửa từ Status -> TrangThai
			}
			members = append(members, member)
		}

		// Thêm owner nếu chưa có trong members
		ownerExists := false
		if room.NguoiTaoID != nil {
			for _, m := range members {
				if id, ok := m["id"].(uint); ok && id == *room.NguoiTaoID {
					ownerExists = true
					break
				}
			}

			if !ownerExists {
				owner := gin.H{
					"id":     room.NguoiTao.ID,
					"name":   room.NguoiTao.Ten,
					"email":  room.NguoiTao.Email,
					"status": "owner",
				}
				members = append([]gin.H{owner}, members...)
			}
		}

		result = append(result, gin.H{
			"id":           room.ID,
			"ten_room":     room.TenRoom,
			"mo_ta":        room.MoTa,
			"trang_thai":   room.TrangThai,
			"member_count": len(members),
			"members":      members,
			"is_public":    room.IsPublic,
		})
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
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
	userVal, exists := c.Get(middleware.CtxUser)
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
// Gửi lời mời
func InviteUserToRoom(c *gin.Context) {
	userVal, exists := c.Get(middleware.CtxUser)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Không tìm thấy user trong context"})
		return
	}
	u := userVal.(models.NguoiDung)

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
		InviterID: u.ID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	if err := config.DB.Create(&invite).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "Không thể gửi lời mời",
			"db_error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Đã gửi lời mời",
		"invite":  invite,
	})
}

// 2. Xem danh sách lời mời trong room
func ListRoomInvites(c *gin.Context) {
	userVal, exists := c.Get(middleware.CtxUser)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userVal.(models.NguoiDung)

	// Lấy query params
	roomIDStr := c.Query("room_id") // room_id là tùy chọn
	status := c.Query("status")     // status là tùy chọn: pending|accepted|rejected

	// Bắt đầu query
	db := config.DB.Model(&models.RoomInvite{}).Where("user_id = ?", user.ID)

	// Lọc theo room nếu có
	if roomIDStr != "" {
		roomIDInt, err := strconv.Atoi(roomIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room_id không hợp lệ"})
			return
		}
		db = db.Where("room_id = ?", uint(roomIDInt))
	}

	// Lọc theo status nếu có
	if status != "" {
		db = db.Where("status = ?", status)
	}

	var invites []models.RoomInvite
	if err := db.Find(&invites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lấy được danh sách lời mời"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invites": invites})
}

// 3. Người dùng phản hồi lời mời (accept / reject)
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
	param := c.Param("id")

	// Lấy user từ context
	userVal, exists := c.Get(middleware.CtxUser)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Không tìm thấy user trong context"})
		return
	}
	user := userVal.(models.NguoiDung)

	var room models.Room
	var err error

	// Hỗ trợ cả ID số và share_url string
	if id, convErr := strconv.ParseUint(param, 10, 64); convErr == nil && id > 0 {
		err = config.DB.Preload("NguoiTao").First(&room, id).Error
	} else {
		// Tìm theo share_url
		err = config.DB.Preload("NguoiTao").Where("share_url = ?", param).First(&room).Error
	}

	if err != nil {
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
		config.DB.Model(&models.RoomNguoiThamGia{}).
			Where("room_id = ? AND nguoi_dung_id = ? AND trang_thai = ?", room.ID, user.ID, "active").
			Count(&isParticipant)

		if room.NguoiTaoID != nil && *room.NguoiTaoID != user.ID && isParticipant == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không có quyền xem danh sách thành viên"})
			return
		}
	}

	// Lấy danh sách participants với thông tin NguoiDung
	var participants []models.RoomNguoiThamGia
	if err := config.DB.
		Preload("NguoiDung").
		Where("room_id = ? AND trang_thai = ?", room.ID, "active").
		Find(&participants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lấy được danh sách thành viên"})
		return
	}

	// Build response với thông tin đầy đủ
	result := make([]gin.H, 0)
	
	ownerID := uint(0)
	if room.NguoiTaoID != nil {
		ownerID = *room.NguoiTaoID
	}

	// Thêm owner vào đầu danh sách
	if ownerID > 0 {
		ownerName := room.NguoiTao.Ten
		if ownerName == "" {
			ownerName = "Chủ phòng"
		}
		result = append(result, gin.H{
			"id":             ownerID, // dùng user_id làm id chính
			"user_id":        ownerID,
			"nguoi_dung_id":  ownerID,
			"ten_nguoi_dung": ownerName,
			"name":           ownerName,
			"email":          room.NguoiTao.Email,
			"status":         "owner",
			"trang_thai":     "owner",
			"is_owner":       true,
		})
	}

	// Thêm các thành viên khác (trừ owner)
	for _, p := range participants {
		// Bỏ qua nếu là owner
		if p.NguoiDungID == ownerID {
			continue
		}

		memberName := p.TenNguoiDung
		if memberName == "" && p.NguoiDung.Ten != "" {
			memberName = p.NguoiDung.Ten
		}
		if memberName == "" {
			memberName = "Người dùng"
		}

		result = append(result, gin.H{
			"id":             p.ID,     // ID của record RoomNguoiThamGia (dùng để xóa)
			"user_id":        p.NguoiDungID,
			"nguoi_dung_id":  p.NguoiDungID,
			"ten_nguoi_dung": memberName,
			"name":           memberName,
			"email":          p.NguoiDung.Email,
			"status":         p.TrangThai,
			"trang_thai":     p.TrangThai,
			"is_owner":       false,
			"ngay_vao":       p.NgayVao,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"room_id":      room.ID,
		"owner": gin.H{
			"id":    ownerID,
			"name":  room.NguoiTao.Ten,
			"email": room.NguoiTao.Email,
		},
		"participants": result,
		"total":        len(result),
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
