## Objektif
Implementasi unit test secara menyeluruh untuk domain User. Tujuannya adalah untuk menguji seluruh fungsionalitas terkait user, meliputi:
- Register
- Login
- Logout
- Update data user
- Mengirim OTP ke email user

## Ruang Lingkup (Scope of Work)
Silakan buat unit test untuk file-file berikut yang berada di dalam direktori `internal/`:
- `domain/user.go`
- `dto/user_dto.go`
- `handler/user_handler.go`
- `repository/user_repo.go`
- `router/router.go`

## Instruksi Tambahan (Penting)
1. **Dilarang mengubah struktur folder atau arsitektur eksisting** yang sudah ada di dalam direktori proyek, terutama di `internal/`.
2. **Dokumentasi API**: Diminta kepada tim terkait agar membuat dokumentasi API menggunakan **Swagger** untuk seluruh endpoint fitur user. Dokumentasi tersebut harap diletakkan pada folder `doc/user/` di *root* project.

## Instruksi Implementasi Unit Test
1. **Branching**: Buat branch baru dengan nama `feature/unit-test-user` dari branch `main`.
2. **Struktur Direktori**: Tempatkan seluruh file test di dalam folder `test/` pada *root* project, dengan mengikuti struktur folder `internal/`:
   ```text
   test/
   └── user/
       ├── domain/
       ├── dto/
       ├── handler/
       └── repository/
   ```
3. **Pull Request**: Setelah implementasi selesai, buat *Pull Request* ke branch `main`. Pastikan semua test berhasil (*pass*) dan tidak ada perubahan struktur folder di luar konteks yang diminta.
