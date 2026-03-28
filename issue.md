## 🎯 Feature: Integrasi Lynk Webhook untuk Otomatisasi Quota (Payment → Credit)

---

## 📌 Overview

aku ingin merubah aplikasi ini berdasarkan quota agar bisa di monetize
Mengimplementasikan integrasi webhook dari Lynk.id untuk otomatis menambahkan quota ke user setelah pembayaran berhasil.

Tujuan:

* agar setiap layanan dibedakan karena api dari ai terbatas
* Memberikan pengalaman seamless: bayar → langsung bisa pakai
* Menjadi fondasi monetisasi SaaS berbasis quota

---

## 🧩 Business Logic

### 💰 Produk di Lynk

| Nama Produk | Harga    | Quiz Quota | Summarize Quota |
| ----------- | -------- | ---------- | --------------- |
| free tier (akun baru untuk promosi) | 0 | 1 | 1 |
| Paket 4x    | Rp10.000 | 4          | 4               |
| Paket 10x   | Rp20.000 | 10         | 10              |

> 📌 Quota bersifat **accumulate** — setiap pembelian menambah quota yang sudah ada, tidak me-reset.

---

### 🔄 Flow Sistem

```
User bayar di Lynk
        ↓
Lynk kirim webhook ke backend
        ↓
Backend verifikasi secret/signature
        ↓
Backend verifikasi payload
        ↓
Idempotency check (transaction_id)
        ↓
Mapping product → quota (quiz + summarize)
        ↓
Update user.quiz_quota + user.summarize_quota
        ↓
Update user.role → "member" (jika sebelumnya "guest")
        ↓
User login → quota & role terupdate otomatis
```

### 🎭 Role Flow

```
Daftar akun baru → role: "guest"  → quiz_quota: 1, summarize_quota: 1
Beli paket di Lynk → role: "member" → quota bertambah sesuai paket
Admin → tidak ada batasan quota
```

---

## 🧱 Backend Implementation

### 1. Endpoint Webhook

```http
POST /webhook/lynk
```

* Public endpoint (tanpa auth JWT)
* Wajib validasi secret

---

### 2. Payload Structure (Expected)

```json
{
  "email": "user@email.com",
  "product_name": "Paket 10x",
  "amount": 20000,
  "status": "success",
  "transaction_id": "abc123"
}
```

> ⚠️ Field actual bisa berbeda — **wajib verifikasi ke docs resmi Lynk sebelum implementasi**:
> - Referensi docs: https://documenter.getpostman.com/view/43601478/2sBXc8o3kn
> - Konfirmasi nama field exak: apakah `product_name`, `item_name`, atau lainnya?
> - Konfirmasi nilai `status`: apakah `"success"`, `"paid"`, `"settlement"`, atau lainnya?
> - Konfirmasi format `transaction_id` dari Lynk

---

### 3. Handler Logic

```go
func HandleLynkWebhook(c *fiber.Ctx) error {
  var payload LynkWebhookPayload

  if err := c.BodyParser(&payload); err != nil {
    return c.SendStatus(400)
  }

  // 1. Validasi status
  if payload.Status != "success" {
    return c.SendStatus(200)
  }

  // 2. Idempotency check
  if isProcessed(payload.TransactionID) {
    return c.SendStatus(200)
  }

  // 3. Mapping produk → quota
  quota := mapProductToQuota(payload.ProductName)

  // 4. Update quota user (quiz + summarize)
  quizQuota, summarizeQuota := mapProductToQuota(payload.ProductName)
  db.Model(&User{}).
    Where("email = ?", payload.Email).
    Updates(map[string]interface{}{
      "quiz_quota":      gorm.Expr("quiz_quota + ?", quizQuota),
      "summarize_quota": gorm.Expr("summarize_quota + ?", summarizeQuota),
    })

  // 5. Update role ke "member" jika masih "guest"
  db.Model(&User{}).
    Where("email = ? AND role = ?", payload.Email, "guest").
    Update("role", "member")

  // 6. Simpan transaksi
  saveTransaction(payload)

  return c.SendStatus(200)
}
```

---

## 🗄️ Database Changes

### 1. Update Table `users`

```sql
ALTER TABLE users ADD COLUMN quiz_quota INT DEFAULT 1;
ALTER TABLE users ADD COLUMN summarize_quota INT DEFAULT 1;
ALTER TABLE users ADD COLUMN role VARCHAR(20) DEFAULT 'guest';
```

> 📌 Free tier diberikan otomatis saat registrasi (`quiz_quota = 1`, `summarize_quota = 1`, `role = "guest"`).
> Saat beli paket, role diupdate ke `"member"` dan quota di-accumulate.

---

### 2. Table: `lynk_transactions`

```sql
CREATE TABLE lynk_transactions (
  id UUID PRIMARY KEY,
  transaction_id VARCHAR UNIQUE,
  email VARCHAR(255),
  product_name VARCHAR(255),
  amount INT,
  status VARCHAR(50),
  created_at TIMESTAMP
);
```

---

## 🔐 Security

### 1. Webhook Secret / Signature Validation

* Set secret di Lynk webhook settings
* Validasi di header

```go
if c.Get("X-Webhook-Secret") != os.Getenv("LYNK_WEBHOOK_SECRET") {
  return c.SendStatus(401)
}
```

> ⚠️ **Perlu dikonfirmasi ke docs Lynk**: apakah menggunakan plain secret di header (`X-Webhook-Secret`)
> atau HMAC-SHA256 signature (seperti `X-Signature`)? Jika HMAC, implementasi berbeda:
> ```go
> // Contoh HMAC jika ternyata Lynk pakai signature
> mac := hmac.New(sha256.New, []byte(os.Getenv("LYNK_WEBHOOK_SECRET")))
> mac.Write(c.Body())
> expected := hex.EncodeToString(mac.Sum(nil))
> if c.Get("X-Signature") != expected {
>   return c.SendStatus(401)
> }
> ```

---

### 2. Idempotency (WAJIB)

Webhook bisa dikirim ulang.

Solusi:

* Gunakan `transaction_id` sebagai unique key
* Jika sudah ada → skip

---

## ⚠️ Edge Cases

### 1. Duplicate Webhook

* Harus aman (tidak double tambah quota)

---

### 2. Email Tidak Ditemukan

Kemungkinan:

* User belum register
* Email berbeda

Solusi:

* Skip + log
* (Future) simpan sebagai pending credit

---

### 3. Mapping Produk Gagal

Jika:

* nama produk berubah di Lynk

Solusi:

* fallback: ignore
* log error

---

### 4. Race Condition

Gunakan atomic update di DB level:

```go
gorm.Expr("quiz_quota + ?", quizQuota)
gorm.Expr("summarize_quota + ?", summarizeQuota)
```

---

### 5. User Belum Daftar Saat Bayar

Kemungkinan:

* User bayar di Lynk sebelum register di aplikasi

Solusi:

* Skip + log warning
* (Future) simpan sebagai **pending credit** — saat user register dengan email sama, quota langsung diberikan

---

## 🧪 Testing Plan

### 1. Local Testing

* Gunakan webhook test dari Lynk
* Expose local server (ngrok)

---

### 2. Test Cases

* [ ] Payment success → quiz_quota bertambah
* [ ] Payment success → summarize_quota bertambah
* [ ] Payment success → role berubah dari "guest" ke "member"
* [ ] Payment success → role "member" tidak berubah jika sudah member
* [ ] Payment failed → tidak ada perubahan quota maupun role
* [ ] Duplicate webhook → tidak double tambah quota
* [ ] Email tidak ditemukan → tidak crash, log warning
* [ ] Produk tidak dikenal → tidak error, log warning
* [ ] Secret/signature salah → return 401
* [ ] Beli paket 2x → quota accumulate (bukan replace)

---

## 📊 Logging

Tambahkan log:

* incoming webhook
* payload
* result (success / skipped / error)

---

## 🚀 Acceptance Criteria

* [ ] Webhook endpoint aktif
* [ ] quiz_quota bertambah otomatis setelah payment
* [ ] summarize_quota bertambah otomatis setelah payment
* [ ] Role otomatis naik dari "guest" ke "member" setelah payment
* [ ] Tidak ada duplicate processing
* [ ] Aman (secret/signature validated)
* [ ] Tidak crash jika data tidak valid
* [ ] User bisa langsung pakai tanpa redeem
* [ ] Free tier quota (1 quiz + 1 summarize) diberikan otomatis saat registrasi

---

## 🧠 Notes

* Gunakan email sebagai identifier utama
* Pastikan user menggunakan email yang sama saat pembayaran dan saat register
* User wajib sudah daftar akun sebelum bayar (untuk MVP)
* Fokus ke reliability, bukan kompleksitas
* Quota bersifat accumulate — pembelian berulang menambah, tidak me-reset
* **Wajib verifikasi payload fields & auth mechanism ke docs Lynk sebelum coding**: https://documenter.getpostman.com/view/43601478/2sBXc8o3kn

---

## 🎭 Role & Quota Rules

| Role   | Quiz Quota  | Summarize Quota | Keterangan                          |
| ------ | ----------- | --------------- | ----------------------------------- |
| guest  | 1 (default) | 1 (default)     | Diberikan saat registrasi           |
| member | sesuai paket (accumulate) | sesuai paket (accumulate) | Naik otomatis setelah beli |
| admin  | unlimited   | unlimited       | Tidak ada batasan akses             |

---

## 📁 Dokumentasi (`doc/`)

Buat file dokumentasi baru di folder `doc/` (sejajar dengan folder `doc/user`, `doc/quiz`, dll):

### File yang perlu dibuat

```
doc/
└── webhook/
    └── lynk.md       ← dokumentasi endpoint webhook Lynk
```

### Isi `doc/webhook/lynk.md`

Dokumentasikan:

* Endpoint: `POST /webhook/lynk`
* Auth: secret/signature validation (tidak pakai JWT)
* Request headers yang dibutuhkan
* Request body (payload dari Lynk) — field, tipe, keterangan
* Response: selalu `200 OK` kecuali secret salah (`401`)
* Business logic: mapping produk → quota + role update
* Contoh payload sukses & gagal
* Referensi: link ke docs Lynk

---

## 📝 README.md Update

Update [README.md](README.md) dengan tambahan berikut:

### 1. Tambah di section Tech Stack / Integrasi

```markdown
| Payment | Lynk.id (webhook) |
```

### 2. Tambah section baru: Quota & Monetisasi

```markdown
## Quota & Monetisasi

Aplikasi menggunakan sistem quota per user untuk membatasi penggunaan layanan AI.

| Role   | Quiz Quota  | Summarize Quota | Cara Mendapat         |
|--------|-------------|------------------|-----------------------|
| guest  | 1           | 1                | Otomatis saat register |
| member | accumulate  | accumulate       | Beli paket di Lynk.id |
| admin  | unlimited   | unlimited        | -                     |

Webhook dari Lynk.id akan otomatis menambah quota dan mengupdate role setelah pembayaran berhasil.
```

### 3. Tambah di section Arsitektur / Flow

Tambahkan node Lynk.id ke diagram arsitektur:

```
User bayar → Lynk.id → POST /webhook/lynk → update quota + role
```

---

## 🚀 Priority

**HIGH — Core monetization feature**
