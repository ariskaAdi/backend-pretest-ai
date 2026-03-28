# Lynk.id Webhook Integration

This document describes the webhook integration with Lynk.id to automate user quotas and roles.

## Endpoint
`POST /api/v1/webhook/lynk`

## Authentication
The endpoint is public but protected by a secret key in the header.
- Header: `X-Webhook-Secret`
- Secret: Defined in environment variable `LYNK_WEBHOOK_SECRET`

## Request Payload (Example)
```json
{
  "email": "user@email.com",
  "product_name": "Paket 10x",
  "amount": 20000,
  "status": "success",
  "transaction_id": "abc123456"
}
```

## Response

Semua response menggunakan format standar `APIResponse`.

### Success
**Status:** `200 OK`
```json
{
  "status": "success",
  "message": "webhook processed successfully",
  "data": null
}
```

### Already Processed (idempotent)
**Status:** `200 OK`
```json
{
  "status": "success",
  "message": "transaksi sudah diproses sebelumnya",
  "data": null
}
```

### Invalid Secret
**Status:** `401 Unauthorized`
```json
{
  "status": "error",
  "message": "invalid webhook secret",
  "data": null
}
```

### Bad Request
**Status:** `400 Bad Request`
```json
{
  "status": "error",
  "message": "format request tidak valid",
  "data": null
}
```

## Environment Setup

Tambahkan ke file `.env`:
```
LYNK_WEBHOOK_SECRET=isi_dengan_secret_dari_lynk_dashboard
```

Cara mendapatkan secret:
1. Login ke Lynk.id dashboard
2. Buka menu Webhook Settings
3. Copy secret key dan paste ke `.env`

## Contoh Payload Non-Success (akan di-skip, tidak update quota)

```json
{
  "email": "user@email.com",
  "product_name": "Paket 4x",
  "amount": 10000,
  "status": "failed",
  "transaction_id": "abc789"
}
```

> Payload dengan `status` selain `"success"` akan diterima dengan `200 OK` tapi tidak memproses quota.

## Testing Lokal

1. Jalankan server lokal
2. Expose via ngrok: `ngrok http 8080`
3. Set webhook URL di Lynk dashboard ke: `https://<ngrok-url>/api/v1/webhook/lynk`
4. Set `X-Webhook-Secret` di Lynk dashboard sama dengan `LYNK_WEBHOOK_SECRET` di `.env`
5. Trigger test payment dari Lynk dashboard

## Business Logic
1. **Verification**: 
   - Checks `X-Webhook-Secret`.
   - Only processes payloads with `"status": "success"`.
2. **Idempotency**: 
   - Uses `transaction_id` to ensure each payment is only processed once.
3. **Quota Mapping**:
   - `Paket 4x` -> +4 Quiz, +4 Summarize
   - `Paket 10x` -> +10 Quiz, +10 Summarize
   - Matching bersifat **case-insensitive**.
4. **Role Upgrade**:
   - Automatically upgrades user from `guest` to `member` upon successful purchase.
5. **Accumulation**:
   - New quotas are added to the existing balance (accumulate).
6. **Atomic Transaction**:
   - Seluruh operasi (simpan history, update quota, update role) dilakukan dalam satu database transaction.

## References
- [Lynk.id API Documentation](https://documenter.getpostman.com/view/43601478/2sBXc8o3kn)
