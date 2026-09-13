# Kế hoạch tổng quát: AuthService làm OIDC Provider cho FileBrowser

## Mục tiêu

Người dùng đăng nhập một lần tại AuthService, chọn FileBrowser từ portal và được FileBrowser tạo session cookie qua OIDC Authorization Code Flow. Không truyền raw access token trên URL và vẫn giữ được phân quyền thư mục qua claim `filebrowserGroups`.

## Hiện trạng đã xác nhận

- AuthService hiện phát hành JWT và có endpoint JWKS tại `/.well-known/jwks.json`.
- AuthService chưa có đầy đủ OIDC discovery, authorization endpoint, token endpoint và userinfo endpoint.
- FileBrowser đã có OIDC client với callback `/api/auth/oidc/callback`.
- FileBrowser đang có cơ chế JWT ngoài và claim `filebrowserGroups`, nhưng đó chỉ là phương án chuyển tiếp, không phải OIDC Authorization Code Flow.
- Kong có thể route các endpoint AuthService và FileBrowser nhưng không tự tạo phiên SSO cho browser.

## Quyết định kiến trúc

AuthService sẽ là OIDC Provider; FileBrowser là OIDC confidential client.

```text
Browser -> AuthService portal
       -> FileBrowser /api/auth/oidc/login
       -> AuthService /oauth2/authorize
       -> AuthService callback/consent
       -> FileBrowser /api/auth/oidc/callback?code=...
       -> AuthService /oauth2/token (server-to-server)
       -> FileBrowser tạo session cookie
```

## Các giai đoạn thực hiện

### 1. Chốt OIDC contract và issuer URL

- Chọn một issuer duy nhất, ví dụ `https://auth.example.com` qua Kong.
- Không dùng đồng thời `localhost` và `127.0.0.1` trong redirect flow.
- Chốt canonical redirect URI:
  `http://127.0.0.1:8085/api/auth/oidc/callback` cho local.
- Chốt claim chuẩn: `sub`, `email`, `preferred_username`, `groups` hoặc `filebrowserGroups`.

**Kết quả:** có tài liệu endpoint/claim/redirect URI làm contract giữa AuthService, Kong và FileBrowser.

### 2. Bổ sung OIDC Provider vào AuthService

Trong `auth-service`:

- Thêm Spring Authorization Server phù hợp với Spring Boot hiện tại.
- Tái sử dụng RSA key pair/JWKS đang dùng cho JWT, nhưng quản lý private key bằng secret/keystore, không commit vào Git.
- Thêm Authorization Server configuration và các endpoint:
  - `/.well-known/openid-configuration`
  - `/oauth2/authorize`
  - `/oauth2/token`
  - `/oauth2/jwks`
  - `/userinfo`
  - `/oauth2/revoke` nếu cần logout/revoke
- Dùng database hiện tại để xác thực user và áp dụng cùng policy account enabled/password/MFA.
- Đăng ký client `filebrowser` với authorization code, PKCE nếu thư viện/client hỗ trợ, client authentication server-side và redirect URI chính xác.
- Phát hành ID Token/UserInfo có `sub`, `email`, `preferred_username`, `filebrowserGroups`.
- Chỉ đưa các department group được phép dùng tenant `filebrowser` vào claim.
- Đưa `filebrowser:admin` cho account có quyền quản trị FileBrowser theo policy đã thống nhất.

**Kết quả:** dùng trình duyệt gọi authorize được và FileBrowser đổi code lấy token thành công.

### 3. Bổ sung session/consent flow cho AuthService UI

Trong `auth-service-ui`:

- Portal không truyền `accessToken` qua query string.
- Card FileBrowser chỉ điều hướng tới:
  `https://filebrowser.example.com/api/auth/oidc/login?redirect=/`
- Nếu AuthService có session cookie thì authorize tự hoàn tất; nếu chưa có thì hiển thị login AuthService.
- Loại bỏ fallback `?jwt=<accessToken>` sau khi OIDC chạy ổn định.
- Logout portal cần cân nhắc logout cả AuthService và FileBrowser qua OIDC logout.

**Kết quả:** click card không làm lộ access token và không cần đăng nhập FileBrowser thủ công.

### 4. Route và bảo mật trên Kong

- Route issuer AuthService và các endpoint OAuth/OIDC qua một host canonical.
- Route FileBrowser callback qua host FileBrowser.
- Không cache `/oauth2/authorize`, `/oauth2/token`, `/userinfo` hoặc callback chứa code.
- Giữ HTTPS ở môi trường thật.
- Cấu hình CORS chỉ cho các origin cần thiết; OAuth redirect không thay thế CORS.
- Không log query parameter `code`, `state`, token hoặc client secret.
- Thêm rate limit cho authorize/token và audit log không chứa credential.

### 5. Cấu hình FileBrowser OIDC

Trong `backend/config.local.yaml` local:

```yaml
auth:
  methods:
    password:
      enabled: false
    oidc:
      enabled: true
      clientID: "filebrowser"
      clientSecret: "${FILEBROWSER_OIDC_CLIENT_SECRET}"
      issuerUrl: "http://localhost:8081"
      scopes: "openid email profile groups"
      userIdentifier: "sub"
      groupsClaim: "filebrowserGroups"
      adminGroup: "filebrowser:admin"
```

- Đăng ký đúng redirect URI với AuthService.
- Giữ source root là `D:/Projects/department-test`.
- Giữ ACL `/accounting` và `/hr` theo group UUID.
- Tắt JWT query handoff sau khi OIDC đã được nghiệm thu.
- Đảm bảo user scope `/` chỉ áp dụng cho admin; user thường dùng scope/ACL theo policy.

### 6. Kiểm thử theo tầng

- Unit: claim mapping, department-to-group mapping, client registration, redirect URI validation.
- Integration: authorize → login → callback → token exchange → userinfo.
- FileBrowser integration: callback tạo cookie, `/api/users?username=self` trả đúng user và groups.
- ACL: user Accounting chỉ thấy `accounting`; user HR chỉ thấy `hr`; user có cả hai group thấy cả hai; admin thấy toàn bộ source.
- Security: code dùng một lần, code hết hạn, sai redirect URI bị từ chối, issuer/audience/signature sai bị từ chối, không chấp nhận raw JWT trong URL.
- E2E qua Kong: portal AuthService → card FileBrowser → dashboard không hiện form login lần hai.

## Acceptance criteria

1. AuthService discovery trả HTTP 200 và mô tả đúng issuer, authorize, token, jwks, userinfo.
2. FileBrowser hoàn tất OIDC callback và tạo cookie session mà không cần password FileBrowser.
3. Không có access token hoặc refresh token trong URL, browser history, access log hoặc referrer.
4. `sub` của user ở AuthService khớp user FileBrowser.
5. `filebrowserGroups` được ánh xạ đúng vào ACL department.
6. Admin thấy toàn bộ `D:/Projects/department-test`; user thường chỉ thấy các folder được cấp quyền.
7. Refresh trang FileBrowser vẫn giữ session; logout rồi truy cập lại phải yêu cầu xác thực AuthService.

## Rủi ro và cách giảm thiểu

- **AuthService chưa phải OIDC Provider:** triển khai theo từng endpoint và test discovery trước.
- **Sai issuer/hostname:** dùng một hostname canonical qua Kong, không trộn localhost/127.0.0.1.
- **Lộ private key/client secret:** dùng environment variable hoặc secret store, không commit.
- **Đồng nhất group không đúng:** định nghĩa một hàm mapping duy nhất từ department membership sang `filebrowserGroups` và test bằng UUID cố định.
- **ACL cũ còn ảnh hưởng:** kiểm tra database FileBrowser sau khi đổi authentication method và test từng loại user.
- **Logout không đồng bộ:** triển khai logout liên service sau khi login flow ổn định.

## Thứ tự bắt đầu đề xuất

1. Kiểm tra version Spring Boot/Spring Security và dependency management của AuthService.
2. Tạo discovery endpoint tối thiểu và test HTTP 200.
3. Tạo client `filebrowser` và Authorization Code Flow nội bộ.
4. Thêm claim `filebrowserGroups` vào ID Token/UserInfo.
5. Cấu hình FileBrowser OIDC và test callback trực tiếp, chưa qua Kong.
6. Đưa flow qua Kong.
7. Xóa fallback truyền `?jwt=` và nghiệm thu ACL.

## ADR

### Decision

AuthService là OIDC Provider trung tâm; FileBrowser dùng OIDC Authorization Code Flow.

### Drivers

- Một lần đăng nhập cho nhiều service.
- Không truyền raw JWT trên URL.
- Giữ phân quyền department hiện có.
- Mở rộng được cho CMS và service khác.

### Alternatives considered

- Query JWT: triển khai nhanh nhưng token có thể lộ qua history/log/referrer.
- One-time custom code: an toàn hơn query JWT nhưng tạo protocol riêng phải bảo trì.
- Keycloak: chuẩn và nhanh hơn nhưng thêm Identity Provider ngoài AuthService.

### Why chosen

OIDC chuẩn giữ AuthService làm trung tâm và cho phép các service tích hợp theo cùng một protocol, thay vì mỗi service phải hiểu custom handshake.

### Consequences

AuthService phải triển khai và vận hành các thành phần OAuth/OIDC quan trọng: authorization code, client registry, signing keys, consent/session và logout.

### Follow-ups

- Quyết định có bắt buộc PKCE cho client FileBrowser hay không.
- Quyết định nơi lưu authorization code/session: database hoặc Redis.
- Xác định logout liên service và rotation RSA keys.

