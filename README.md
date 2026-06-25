# Classroom Attendance and Cognitive Analysis System

CPU-first SaaS attendance management platform with admin and teacher roles.

- React + Vite provides the SaaS UI, webcam face enrollment, and live attendance sampling.
- Go + Gin is the primary API, using GORM repositories/services/controllers, JWT auth, bcrypt password hashes, role permissions, a 4-worker frame pool, and an in-memory face cache.
- Python + FastAPI runs CPU-only ONNX Runtime for emotion/state, gaze, and liveness analysis.
- PostgreSQL stores organizations, users, academics, students, enrollments, cameras, attendance sessions/events, manual corrections, audit logs, and cognitive snapshots.

No raw PyTorch or TensorFlow runtime is used.

## Project Layout

```text
cognitive-service/
  app/
  scripts/
  tests/
go-backend/
  cmd/server/
  internal/config/
  internal/controllers/
  internal/database/
  internal/middleware/
  internal/models/
  internal/repositories/
  internal/security/
  internal/services/
  internal/worker/
  migrations/
frontend/
  src/app/
  src/components/
  src/config/
  src/hooks/
  src/pages/
  src/services/
  src/styles/
docker-compose.yml
```

## Main Features

- First-time admin setup, admin login, teacher login, current profile check.
- JWT bearer auth with role-based backend permissions.
- Passwords stored as bcrypt hashes.
- Organization-scoped data model.
- Admin management for teachers, students, classes, subjects, teacher assignments, sessions, cameras, enrollment, recognition simulation, and manual corrections.
- Teacher accounts have read/attendance access without admin mutation routes.
- Teacher rows show name, email, enrollment status, pose count, and account status.
- Student rows show name, student ID, class/section, consent status, and pose count.
- Webcam enrollment for students and teachers with five required poses: front, left, right, look up, look down.
- Attendance dashboard with totals, review queue, weekly operations, attendance mix, class distribution, subject coverage, active sessions, and teacher workload.
- Attendance page supports session creation, live webcam recognition sampling, events, correction, recognition simulation, and camera records.

## CPU Performance Controls

- Live frame sampling: `frontend/src/hooks/useFrameSampler.js` sends one JPEG frame every `VITE_SAMPLE_INTERVAL_MS`, default `3000`.
- Enrollment capture is pose-by-pose, not a continuous upload stream.
- Go worker pool: `MAX_FRAME_WORKERS=4` limits concurrent frame processing.
- Go queue backpressure: `FRAME_QUEUE_SIZE=16` rejects excess frames.
- In-memory face cache: Go loads active `student_face_embeddings` at startup and matches in RAM.
- ONNX Runtime CPU provider only: the Python service creates sessions with `CPUExecutionProvider`.

## Database

The Go backend uses GORM AutoMigrate plus SQL migrations in `go-backend/migrations`.

Expected local variables:

```env
DB_CONNECTION=pgsql
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=attendance
DB_USERNAME=attendance
DB_PASSWORD=
```

If your local PostgreSQL role shows `role "attendance" is not permitted to log in`, fix the role in PostgreSQL:

```powershell
psql -U postgres -d postgres
ALTER ROLE attendance WITH LOGIN;
CREATE DATABASE attendance OWNER attendance;
\q
```

Docker users can start the bundled PostgreSQL:

```powershell
docker compose up -d postgres
```

## Python Cognitive Service

```powershell
cd cognitive-service
py -3.12 -m venv .venv
.\.venv\Scripts\Activate.ps1
python -m pip install -r requirements.txt
python scripts/create_dummy_onnx_models.py
python -m uvicorn app.main:app --host 0.0.0.0 --port 8001
```

The included dummy ONNX files are tiny CPU-only development models. Replace them with real INT8 models by setting:

```powershell
$env:EMOTION_MODEL_PATH="models/emotion_int8.onnx"
$env:GAZE_MODEL_PATH="models/gaze_int8.onnx"
$env:LIVENESS_MODEL_PATH="models/liveness_int8.onnx"
```

## Go Backend

```powershell
cd go-backend
copy .env.example .env
go mod download
go run ./cmd/server
```

If PowerShell says `go` is not recognized but Go is installed in the default Windows location:

```powershell
& "C:\Program Files\Go\bin\go.exe" mod download
& "C:\Program Files\Go\bin\go.exe" run ./cmd/server
```

`FACE_RECOGNIZER_MODE=mock` is the default CPU-safe demo path. It keeps the app runnable without native dlib assets. For real dlib/go-face recognition:

```powershell
$env:FACE_RECOGNIZER_MODE="dlib"
$env:DLIB_MODEL_DIR="C:\models\dlib"
go run -tags dlib ./cmd/server
```

Face enrollment supports `FACE_EMBEDDING_MODEL_PATH=models/face_embedding.onnx`. If that file is missing, the app stores deterministic demo embeddings for UI/backend testing only, not real biometric recognition.

## React Frontend

```powershell
cd frontend
npm install
npm run dev
```

Open the Vite URL, run first-time admin setup, then create teachers, students, classes, subjects, assignments, and attendance sessions.

## Docker

```powershell
docker compose up --build
```

## API Summary

Auth:

- `GET /api/auth/setup-status`
- `POST /api/auth/setup`
- `POST /api/auth/login`
- `GET /api/auth/me`

Main API:

- `GET /api/dashboard`
- `GET|POST /api/teachers`
- `GET|POST /api/students`
- `GET|POST /api/academics/classes`
- `GET|POST /api/academics/subjects`
- `GET|POST /api/academics/assignments`
- `GET|POST /api/attendance/sessions`
- `GET /api/attendance/events`
- `POST /api/attendance/simulate`
- `POST /api/attendance/events/:id/corrections`
- `GET|POST /api/cameras`
- `POST /api/enrollments/students`
- `POST /api/enrollments/teachers`
- `POST /api/frames`

Python:

- `GET /health`
- `POST /analyze` multipart field `file`

## Verification

```powershell
cd cognitive-service
py -3.12 -m pytest

cd ..\frontend
npm run build

cd ..\go-backend
go test ./...
```
