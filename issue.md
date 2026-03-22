# 📋 ISSUE: Unit Test — Quiz Case

## Status
`open`

## Priority
`high`

## Assignee
_unassigned_

---

## Background & Alur Quiz Case

Quiz case bergantung pada dua hal: **summary modul yang sudah ada** (hasil proses AI dari case module) dan **Genkit service** untuk generate soal. Berikut alur lengkapnya sebelum mulai test.

---

### Alur 1 — Generate Quiz `POST /api/v1/quiz`

```
Client kirim:
{
    "module_id": "uuid",
    "num_questions": 10   ← hanya boleh 5, 10, atau 20
}

QuizService.Generate()
    ├── 1. FindByID(module_id)
    │       ├── nil          → ErrModuleNotFound
    │       └── UserID != userID → ErrNotModuleOwner
    │
    ├── 2. Cek module.IsSummarized == true && summary != ""
    │       └── false → ErrModuleNotSummarized (summary AI belum selesai)
    │
    ├── 3. pkgai.Client.GenerateQuiz(module.Summary, numQuestions)
    │       └── HTTP POST ke Genkit :3400/generateQuiz
    │               └── Gemini generate soal dari summary
    │               └── return []Question{text, options[4], answer}
    │
    ├── 4. Marshal options ke JSON string (disimpan di DB sebagai JSONB)
    │
    ├── 5. quizRepo.Create() → simpan quiz + questions ke DB
    │       └── status: "pending", score: null
    │
    └── return QuizResponse
            └── berisi soal TANPA correct_answer (tidak bocor ke client)
```

---

### Alur 2 — Submit Jawaban `POST /api/v1/quiz/:id/submit`

```
Client kirim:
{
    "answers": [
        {"question_id": "uuid", "answer": "A"},
        {"question_id": "uuid", "answer": "C"},
        ...
    ]
}

QuizService.Submit()
    ├── 1. FindByID(quizID) → cek exist + ownership
    ├── 2. Cek status != "completed" → ErrQuizAlreadyDone
    ├── 3. Cek len(answers) == len(questions) → ErrAnswerCountMismatch
    ├── 4. Build answerMap: questionID → jawaban user
    ├── 5. Loop tiap question:
    │       ├── ambil jawaban dari answerMap
    │       │       └── tidak ketemu → ErrInvalidQuestionID
    │       ├── set question.UserAnswer = answer
    │       └── kalau answer == correct_answer → correct++
    │
    ├── 6. score = (correct * 100) / total_soal
    │
    ├── 7. quizRepo.SaveAnswersAndScore() — dalam 1 DB transaction:
    │       ├── UPDATE questions SET user_answer = ? WHERE id = ?  (per soal)
    │       └── UPDATE quizzes SET score = ?, status = 'completed'
    │
    └── return QuizResultResponse
            └── berisi soal + correct_answer + user_answer + is_correct per soal
```

---

### Alur 3 — Retry Quiz `POST /api/v1/quiz/:id/retry`

```
QuizService.Retry()
    ├── 1. FindByID(quizID) → ambil module_id dan num_questions dari quiz lama
    ├── 2. Validasi ownership
    └── 3. Panggil Generate() dengan module_id + num_questions yang sama
            └── AI generate soal BARU dari summary yang sama
            └── Simpan sebagai quiz baru (quiz lama tetap ada di history)
```

> Retry tidak menghapus quiz lama. Quiz baru dibuat terpisah sehingga history tetap lengkap.

---

### Alur 4 — History `GET /api/v1/quiz/history`

```
QuizService.GetHistory()
    └── FindByUserID() → semua quiz user, ORDER BY created_at DESC
            └── return []QuizHistoryResponse
                    ├── score: null  (kalau status "pending")
                    └── score: 80   (kalau status "completed")
```

---

### Alur 5 — History by Module `GET /api/v1/quiz/history/module/:moduleId`

```
QuizService.GetHistoryByModule()
    └── FindByUserIDAndModuleID() → filter quiz berdasarkan modul tertentu
            └── berguna untuk lihat progress retry di satu modul
```

---

### Alur 6 — Lihat Hasil `GET /api/v1/quiz/:id/result`

```
QuizService.GetResult()
    ├── FindByID() → cek exist + ownership
    └── return QuizResultResponse (sama seperti response Submit)
            └── bisa dipanggil berulang kali setelah quiz selesai
```

---

## Struktur DB

```
quizzes
    id, user_id, module_id, num_questions, score (null/int), status (pending/completed)

questions
    id, quiz_id, text, options (JSONB), correct_answer, user_answer (null saat pending)
```

---

## Task 1 — Interface sudah ada, pastikan konsisten

Interface `QuizServiceContract` dan `QuizRepositoryContract` **sudah didefinisikan** masing-masing di:
- `internal/service/quiz_service.go` → `QuizServiceContract`
- `internal/repository/quiz_repo.go` → `QuizRepositoryContract`

Pastikan `QuizHandler` menggunakan `QuizServiceContract` (bukan struct konkret `*QuizService`). Sudah benar di implementasi saat ini.

---

## Task 2 — Unit Test: `test/quiz/service/quiz_service_test.go`

### Setup mock:

```go
type MockQuizRepository struct {
    mock.Mock
}
// implement semua method QuizRepositoryContract

type MockModuleRepository struct {
    mock.Mock
}
// implement semua method ModuleRepositoryContract
```

### Test case `Generate()`:

| # | Skenario | Expected |
|---|---|---|
| 1 | Modul tidak ditemukan | `ErrModuleNotFound` |
| 2 | Modul milik user lain | `ErrNotModuleOwner` |
| 3 | `is_summarized = false` | `ErrModuleNotSummarized` |
| 4 | Summary kosong meski `is_summarized = true` | `ErrModuleNotSummarized` |
| 5 | Genkit gagal (error) | return error |
| 6 | Sukses, `num_questions=5` | return `QuizResponse` dengan 5 soal, status `pending` |
| 7 | Sukses, `num_questions=10` | return `QuizResponse` dengan 10 soal |
| 8 | Response soal tidak mengandung `correct_answer` | field `correct_answer` tidak ada di `QuestionResponse` |

### Test case `Submit()`:

| # | Skenario | Expected |
|---|---|---|
| 1 | Quiz tidak ditemukan | `ErrQuizNotFound` |
| 2 | Quiz milik user lain | `ErrNotQuizOwner` |
| 3 | Quiz sudah `completed` | `ErrQuizAlreadyDone` |
| 4 | Jumlah jawaban kurang | `ErrAnswerCountMismatch` |
| 5 | Jumlah jawaban lebih | `ErrAnswerCountMismatch` |
| 6 | `question_id` tidak valid | `ErrInvalidQuestionID` |
| 7 | Semua jawaban benar | `score = 100` |
| 8 | Semua jawaban salah | `score = 0` |
| 9 | Sebagian benar (6/10) | `score = 60` |
| 10 | `SaveAnswersAndScore` DB error | return error |

### Test case `GetHistory()`:

| # | Skenario | Expected |
|---|---|---|
| 1 | User belum punya quiz | return slice kosong |
| 2 | User punya mix pending + completed | return semua, score null untuk pending |
| 3 | DB error | return error |

### Test case `GetHistoryByModule()`:

| # | Skenario | Expected |
|---|---|---|
| 1 | Tidak ada quiz untuk modul itu | return slice kosong |
| 2 | Ada beberapa quiz untuk modul itu | return hanya quiz dari modul tersebut |

### Test case `GetResult()`:

| # | Skenario | Expected |
|---|---|---|
| 1 | Quiz tidak ditemukan | `ErrQuizNotFound` |
| 2 | Quiz milik user lain | `ErrNotQuizOwner` |
| 3 | Sukses | return `QuizResultResponse` dengan detail jawaban |

### Test case `Retry()`:

| # | Skenario | Expected |
|---|---|---|
| 1 | Quiz lama tidak ditemukan | `ErrQuizNotFound` |
| 2 | Quiz lama milik user lain | `ErrNotQuizOwner` |
| 3 | Modul sudah tidak tersummarisasi | `ErrModuleNotSummarized` |
| 4 | Sukses | return `QuizResponse` baru, quiz lama tetap ada |
| 5 | Sukses | `module_id` dan `num_questions` sama dengan quiz lama |

---

## Task 3 — Unit Test: `test/quiz/handler/quiz_handler_test.go`

Gunakan `app.Test()` dari Fiber. Mock `QuizServiceContract`.

| Endpoint | Skenario | Expected HTTP |
|---|---|---|
| `POST /quiz` | Body invalid | `400` |
| `POST /quiz` | `num_questions` bukan 5/10/20 | `400` |
| `POST /quiz` | Modul tidak ditemukan | `404` |
| `POST /quiz` | Modul belum disummarisasi | `400` |
| `POST /quiz` | Sukses | `201` + quiz data |
| `POST /quiz/:id/submit` | Jawaban tidak lengkap | `400` |
| `POST /quiz/:id/submit` | Quiz sudah dikerjakan | `400` |
| `POST /quiz/:id/submit` | Sukses | `200` + result |
| `POST /quiz/:id/retry` | Quiz tidak ditemukan | `404` |
| `POST /quiz/:id/retry` | Sukses | `201` + quiz baru |
| `GET /quiz/history` | Sukses | `200` + list |
| `GET /quiz/history/module/:id` | Sukses | `200` + list |
| `GET /quiz/:id/result` | Quiz tidak ditemukan | `404` |
| `GET /quiz/:id/result` | Sukses | `200` + result |

---

## Struktur File

```
test/
├── quiz/
│   ├── service/quiz_service_test.go      ← buat baru
│   └── handler/quiz_handler_test.go      ← buat baru
```

---

## Task 4 — dokumentasi: `doc/quiz/swagger.yml`

-- buat dokumentasi di folder doc/quiz/swagger.yml 


## Catatan Penting

- **Jangan connect ke DB sungguhan** — semua pakai mock
- **Jangan call Genkit sungguhan** — mock `pkgai.Client.GenerateQuiz()`
- **Jangan ubah file existing jika tidak ada konfirmasi dan tidak ada di issue**
- **Buat branch baru dengan nama feature/quiz-test lalu lakukan PR ke branch main**
- Test `Submit()` harus memastikan kalkulasi skor benar secara matematis
- Test `Retry()` harus memastikan quiz lama tidak terhapus (repo `FindByID` quiz lama masih ada)
- `correct_answer` tidak boleh muncul di `QuestionResponse` (hanya di `QuestionResultResponse`)

```bash
go test ./internal/service/... -v -race
go test ./internal/handler/... -v
go test ./... -cover
```

---

## Ekspektasi Coverage

| Package | Target |
|---|---|
| `internal/service/quiz_service.go` | ≥ 80% |
| `internal/handler/quiz_handler.go` | ≥ 60% |

---

## Definition of Done

- [ ] Semua test case di tabel atas diimplementasi
- [ ] Kalkulasi skor diverifikasi dengan test case benar/salah/campuran
- [ ] `correct_answer` dipastikan tidak bocor di response generate/retry
- [ ] `go test ./... -race` lulus tanpa error
- [ ] Coverage `quiz_service` minimal 80%
