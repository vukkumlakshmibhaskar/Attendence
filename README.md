# Classroom Attendance and Cognitive Analysis System

CPU-first SaaS attendance management and cognitive monitoring platform with admin and teacher roles.

---

## Project Structure (2 Folders)

```text
Attendence/
├── backend/
│   ├── server.py              # Unified FastAPI backend (SQLite, Auth, Face Recognition, Cognitive AI)
│   ├── requirements.txt       # Python dependencies
│   ├── index.html             # Bundled standalone web application
│   ├── cognitive-service/     # ONNX cognitive models and training scripts
│   ├── go-backend/            # Go backend implementation & SQL migrations
│   └── docs/                  # Architecture & model strategy documentation
│
└── frontend/
    ├── src/                   # React + Vite application source
    ├── standalone.html        # Single-file zero-build standalone HTML interface
    ├── package.json           # Frontend dependencies (React 19, Vite 6, MediaPipe)
    └── dist/                  # Production compiled build
```

---

## Running the Project

### 1. Start the Backend
Open a terminal in `backend/`:

```powershell
cd backend
python server.py
```
> The backend automatically starts on **`http://localhost:8080`** with embedded SQLite database, in-memory face cache, and all REST endpoints.

### 2. Start the Frontend
You have two options:

#### Option A: React Development Server (Vite)
Open a new terminal in `frontend/`:

```powershell
cd frontend
npm install
npm run dev
```
> Open **`http://localhost:5173`** in your browser. It will automatically connect to the backend on `http://localhost:8080`.

#### Option B: Zero-Build Standalone Interface
Simply open **`http://localhost:8080`** in your browser while `python backend/server.py` is running! The backend directly serves the UI.

---

## Core Features
- **First-time Admin Setup & Login**: Complete organization onboarding with JWT authentication and bcrypt/PBKDF2 passwords.
- **Role-Based Access**: `admin` can manage teachers, students, classes, subjects, and cameras; `teacher` can conduct attendance.
- **Guided 5-Pose Face Enrollment**: Webcam module enforcing 5 poses (*Front, Left, Right, Look Up, Look Down*) to generate robust 128-dimensional face vectors.
- **Live Recognition & Attendance**: Real-time webcam sampling with face bounding-box overlays, matched student/faculty details, and confidence scores.
- **Cognitive Telemetry**: Real-time Emotion (*Attentive, Tired, Distracted*), Gaze (*Screen, Away*), and Anti-Spoof Liveness (*Live, Spoof*).
- **Interactive Dashboard**: KPI counters, attendance mix distribution, weekly operations trend, and class breakdowns.
