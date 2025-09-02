# Survey Server

## Giới thiệu

Đây là dự án Survey Server sử dụng Golang, Gin framework và PostgreSQL.

## Yêu cầu

- Docker & Docker Compose
- Git

## Cài đặt

1. **Clone repository:**

   ```sh
   git clone https://github.com/vnkhanh/survey-server.git
   cd survey-server
   ```

2. **Cấu hình biến môi trường:**

  Copy file `.env.example` thành `.env`:
   ```bash
    cp .env.example .env    # Linux/macOS
    copy .env.example .env  # Windows
   ```

4. **Chạy ứng dụng bằng Docker Compose:**

   ```sh
   docker-compose up --build
   ```

5. **Truy cập server:**

   - Mở trình duyệt và truy cập: [http://localhost:8081/ping](http://localhost:8080/ping)
   - Nếu thành công sẽ nhận được:
     ```json
     {"message": "pong"}
     ```
   - Hoặc [http://localhost:8081](http://localhost:8081)
   - Nếu thành công:
      ```raw
     Survey server is running
     ```

## Cấu trúc thư mục
Một số thư mục không hiển thị ở trên là do còn rỗng nên không push được
- `cmd/`: Chứa file main khởi động server
- `config/`: Cấu hình ứng dụng
- `controllers/`: Xử lý logic các API
- `middlewares/`: Các middleware cho Gin
- `models/`: Định nghĩa các model
- `routes/`: Định nghĩa các route
- `services/`: Xử lý nghiệp vụ, các chức năng liên quan đến logic chính của ứng dụng
- `utils/`: Các hàm tiện ích

## Quy trình làm việc nhóm với Git

1. **Tạo branch mới cho mỗi tính năng hoặc sửa lỗi:**

   ```sh
   git checkout -b ten-branch-moi
   ```

2. **Làm việc và commit code:**

   ```sh
   git add .
   git commit -m "Mô tả ngắn gọn về thay đổi"
   ```

3. **Đẩy branch lên GitHub:**

   ```sh
   git push origin ten-branch-moi
   ```

4. **Tạo Pull Request (PR):**
   - Vào GitHub, chọn branch vừa push và tạo Pull Request để xin review.

5. **Review và hợp nhất (merge) code:**
   - Thành viên khác review, sau đó leader hoặc người được phân công sẽ merge vào branch `master` hoặc `main`.

6. **Luôn cập nhật branch trước khi merge:**

   ```sh
   git checkout main
   git pull origin main
   git checkout ten-branch-moi
   git merge main
   ```
---

**Lưu ý:**  
- Không commit trực tiếp lên branch `master`.
- `master` là chính còn `main` là test nên có gì thì push vào `main`.
