# Classroom Attendance and Cognitive Analysis System

CPU-first SaaS attendance management and cognitive monitoring platform with admin and teacher roles.

---

## Project Structure (2 Folders)

```text
Attendence/
├── backend/
│   ├── server.py              # Unified FastAPI backend (PostgreSQL, Auth, Face Recognition, Cognitive AI)
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

## Running the Project (Windows)

From the project root, run:

```powershell
.\start_all.bat
```

This starts the existing local PostgreSQL database, backend, and frontend.
Open **http://127.0.0.1:5174** for Attendance; port 5173 is used by another local project.
The API is at **http://127.0.0.1:8080**. Re-running the launcher reuses its running services.
Use `.\stop_all.bat` to stop this project's services.

For manual backend development, run these commands from `backend/` after stopping
the launcher's backend (two servers cannot share port 8080):

```powershell
python -m pip install -r requirements.txt
python setup_local_db.py
python -m uvicorn server:app --host 127.0.0.1 --port 8080 --reload
```

In a separate terminal, run `npm run dev` from `frontend/`.
The backend reads `backend/.env` before loading database settings, including
`DB_PORT=5433` and `DB_USERNAME=attendance`; it does not use the system PostgreSQL
credentials on port 5432. Keep the ignored `.env` and `postgres-data/` between runs.

If these fixes disappear after a Git operation, check `git status` and `git stash list`:
startup, login, and enrollment changes must be present in the working tree to take effect.
Do not blindly apply a stash over newer changes, since that can create merge markers.

---

## Core Features
- **First-time Admin Setup & Login**: Complete organization onboarding with JWT authentication and bcrypt/PBKDF2 passwords.
- **Role-Based Access**: `admin` can manage teachers, students, classes, subjects, and cameras; `teacher` can conduct attendance.
- **Guided 5-Pose Face Enrollment**: Webcam module enforcing 5 poses (*Front, Left, Right, Look Up, Look Down*) to generate robust 128-dimensional face vectors.
- **Live Recognition & Attendance**: Real-time webcam sampling with face bounding-box overlays, matched student/faculty details, and confidence scores.
- **Cognitive Telemetry**: Real-time Emotion (*Attentive, Tired, Distracted*), Gaze (*Screen, Away*), and Anti-Spoof Liveness (*Live, Spoof*).
- **Interactive Dashboard**: KPI counters, attendance mix distribution, weekly operations trend, and class breakdowns.


## Local configuration (Windows)

Run `start_all.bat` to start the project database, Python API and React frontend.
Run `stop_all.bat` to stop those project processes. Logs are in `runtime-logs/`.
Install dependencies with `python -m pip install -r backend/requirements.txt` and
`cd frontend && npm install` before the first launch on another machine.

The Python backend loads `backend/.env`. The separate Go implementation's
`backend/go-backend/.env` is not used by this launcher. The local PostgreSQL 18
cluster is in `postgres-data/`, bound to `127.0.0.1:5433`, with database and user
`attendance`. Credentials are stored only in the ignored `backend/.env`.
This leaves any system PostgreSQL instance on port 5432 independent.

`FACE_VERIFIER_MODEL_PATH=face_verifier_pins.json` selects the supplied SFace
verifier trained for 300 epochs on 105_classes_pins_dataset. Its recorded test
accuracy is 96.62%, precision 98.90%, recall 94.20%, and raw cosine threshold
0.277. Runtime uses the trained logistic probability with decision threshold
0.5; it does not substitute the raw cosine threshold for that probability.
Both enrollment and recognition use YuNet detection and normalized SFace
128-dimensional embeddings. Old grid enrollments are excluded from the cache
and must be enrolled again. Enrollment requires exactly one detected face.
The `/health` endpoint checks the database and reports the active model.

Open http://localhost:5174 and complete first-time admin setup. Training data
does not enroll students automatically. Enroll each student/teacher in the app.
The Python cognitive outputs (emotion, gaze and liveness) remain heuristics.
