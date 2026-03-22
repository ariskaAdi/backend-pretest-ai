# 📋 ISSUE: Swagger API Documentation & Test Refactoring — Module Case

## Status
`open`

## Priority
`high`

## Assignee
_unassigned_

---

## Aturan Pengerjaan (PENTING)
Tim yang mengerjakan task ini **DIWAJIBKAN** membuat branch baru dari `main` dengan nama:
`feature/module-doc-test`

Seluruh komit harus dilakukan di dalam branch tersebut sebelum di-Push dan dibuatkan Pull Request (PR).

---

## Background

Kita telah berhasil menambahkan *Unit Test* yang komprehensif pada fitur Module. Namun, saat ini file test tersebut (`module_service_test.go` dan `module_handler_test.go`) tanpa sengaja tercampur di dalam folder `test/user`. Selain itu, kita juga membutuhkan dokumentasi API (Swagger) yang jelas bagi tim Frontend/Client. 

Tugas ini dibagi menjadi dua bagian: **Pembuatan Dokumentasi Swagger** dan **Perbaikan Struktur Folder Test**.

---

## Task 1 — Refactor Struktur Folder Test (Modularisasi)

Agar struktur testing lebih rapi dan modular, file test untuk domain *Module* harus memiliki domain foldernya sendiri, tidak boleh bercampur dengan domain *User*.

1. Buat struktur folder baru di dalam `/test`:
   ```
   test/
   └── module/
       ├── handler/
       └── service/
   ```
2. Pindahkah file test terkait module yang ada di dalam `test/user` ke folder baru tersebut:
   - Pindahkan `module_service_test.go` (beserta mock) ke `test/module/service/module_service_test.go`
   - Pindahkan `module_handler_test.go` (beserta mock) ke `test/module/handler/module_handler_test.go`
3. Pastikan `package` logis mengikuti nama foldernya, dan jalankan perintah test untuk memastikan tidak ada yang *broken* sesudah dipindahkan:
   ```bash
   go test ./test/module/... -v 
   ```

---

## Task 2 — Tambahan Komentar Godoc (Swagger)

Di dalam file `internal/handler/module_handler.go`, tambahkan deklarasi *annotation* rutin wajib Swagger di atas setiap deklarasi handler.

Seluruh endpoint Module membutuhkan autentikasi (menggunakan token JWT), sehingga wajib menyertakan tag `@Security BearerAuth`.

### Daftar Endpoint yang Perlu Didokumentasikan:

1. **POST /modules** (`Upload`)
   - **Summary**: Upload a new module PDF
   - **Description**: Upload a PDF file to be parsed and summarized asynchronously.
   - **Accept**: multipart/form-data
   - **Params**:
     - `title` (formData, required) — Judul dari modul.
     - `file` (formData, required) — File dokumen berekstensi `.pdf`.
   - **Responses**: `201 Created` (return `ModuleResponse`), `400 Bad Request`, `500 Server Error`.

2. **GET /modules** (`GetAll`)
   - **Summary**: Get all modules
   - **Description**: Get all modules belonging to the authenticated user.
   - **Responses**: `200 OK` (return array of `ModuleResponse`), `500 Server Error`.

3. **GET /modules/:id** (`GetByID`)
   - **Summary**: Get module details
   - **Description**: Get specific module details including its AI summary.
   - **Params**: `id` (path, required) — UUID dari modul terkait.
   - **Responses**: `200 OK` (return `ModuleDetailResponse`), `401 Unauthorized`, `404 Not Found`, `500 Server Error`.

4. **DELETE /modules/:id** (`Delete`)
   - **Summary**: Delete a module
   - **Description**: Delete a specific module and its history.
   - **Params**: `id` (path, required) — UUID dari modul terkait.
   - **Responses**: `200 OK`, `401 Unauthorized`, `404 Not Found`, `500 Server Error`.

---

## Task 3 — Generate Swagger Docs

Setelah semua komentar di-setup, tim perlu men-generate dokumentasi ini ke dalam folder spesifik `doc/module/`.

Jalankan perintah berikut di terminal pada root project:
```bash
swag init -g internal/router/router.go -o doc/module/ --parseDependency --parseInternal
```

Pastikan struktur direktori nantinya akan menjadi seperti ini:
```
doc/
├── user/
│   ├── swagger.json
│   └── swagger.yaml
└── module/
    ├── swagger.json
    └── swagger.yaml
```

---

## Definition of Done

- [ ] Pengerjaan dilakukan pada branch `feature/module-doc-test`.
- [ ] File test module telah dipindahkan ke folder `test/module/service/` dan `test/module/handler/` dengan namespace package yang benar.
- [ ] Menjalankan `go test ./test/module/...` berjalan 100% tanpa error.
- [ ] Seluruh endpoint (`Upload`, `GetAll`, `GetByID`, `Delete`) di `module_handler.go` memiliki komentar Godoc yang sesuai.
- [ ] File swagger (`swagger.json`, `swagger.yaml`, `docs.go`) berhasil digenerate di dalam folder `doc/module/`.
