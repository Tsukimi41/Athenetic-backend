# Athenetic Backend Architecture & API Reference

**Reference**: See [PRODUCT_SPECIFICATION.md](../PRODUCT_SPECIFICATION.md) for full product vision.

---

## 1. ARCHITECTURE OVERVIEW

### 1.1 Tech Stack
```
Language:      Go 1.21+
Framework:     Echo v4 (lightweight, high-performance)
ORM:           GORM v2 (type-safe queries, migrations)
Database:      PostgreSQL 15 (Docker)
Authentication: JWT (planned; not yet implemented)
Logging:       slog (Go 1.21 structured logging)
Testing:       testing + testify
```

### 1.2 Project Structure
```
Athenetic-backend/
├── cmd/
│   └── server/
│       └── main.go           # Entry point, server setup
├── internal/
│   ├── database/
│   │   ├── db.go             # Connection, migrations
│   │   └── models.go         # ORM models
│   ├── handlers/
│   │   ├── workout.go        # Workout endpoints
│   │   ├── analytics.go      # Volume analytics
│   │   ├── readiness.go      # Autoregulation endpoints
│   │   └── nutrition.go      # Nutrition (future)
│   ├── models/
│   │   ├── workout.go        # Request/response DTOs
│   │   ├── analytics.go      # Analytics models
│   │   └── errors.go         # Error structs
│   ├── routes/
│   │   └── routes.go         # Route registration
│   ├── middleware/
│   │   ├── auth.go           # JWT validation (future)
│   │   └── cors.go           # CORS headers
│   └── utils/
│       ├── calculations.go   # Progressive overload algo
│       └── validators.go     # Input validation
├── migrations/               # Database migrations
├── docker-compose.yml
├── go.mod
└── README.md
```

---

## 2. DATABASE SCHEMA

### 2.1 Core Tables

#### users
```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255),              -- JWT auth (future)
  body_weight_kg DECIMAL(5,2),             -- For protein calc
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

#### exercises
```sql
CREATE TABLE exercises (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  muscle_group VARCHAR(50) NOT NULL,      -- CHEST, BACK, LEGS, SHOULDERS
  default_target_sets INT DEFAULT 3,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Pre-seed:
INSERT INTO exercises (name, muscle_group) VALUES
  ('Barbell Bench Press', 'CHEST'),
  ('Dumbbell Flyes', 'CHEST'),
  ('Barbell Rows', 'BACK'),
  ('Pull-ups', 'BACK'),
  ('Leg Press', 'LEGS'),
  ('Deadlifts', 'LEGS');
```

#### training_sessions
```sql
CREATE TABLE training_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  session_date DATE NOT NULL,
  muscle_group VARCHAR(50),                -- Primary muscle today
  readiness_score INT,                    -- 0–100 (auto-calculated)
  notes TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_date ON training_sessions(user_id, session_date);
```

#### sets
```sql
CREATE TABLE sets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES training_sessions(id) ON DELETE CASCADE,
  exercise_id UUID NOT NULL REFERENCES exercises(id),
  weight_kg DECIMAL(6,2) NOT NULL,        -- Barbell weight (or bodyweight)
  reps_completed INT NOT NULL,            -- Actual reps achieved
  rpe INT,                                 -- Rate of Perceived Exertion (1–10)
  rir INT,                                 -- Reps In Reserve
  target_reps_next INT,                   -- Auto-calculated next target
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sets_exercise_date 
  ON sets(exercise_id, created_at);
```

#### daily_readiness_inputs (New in Phase 1)
```sql
CREATE TABLE daily_readiness_inputs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  input_date DATE NOT NULL,
  sleep_hours DECIMAL(3,1),                -- 0–12
  muscle_soreness INT,                    -- 0–10
  running_km_prior_day DECIMAL(5,2),      -- Interference effect
  readiness_score INT,                    -- Calculated result
  deload_factor DECIMAL(3,2),              -- 0.7–1.0
  created_at TIMESTAMP DEFAULT NOW(),
  
  UNIQUE(user_id, input_date)
);
```

#### nutrition_logs (New in Phase 3)
```sql
CREATE TABLE nutrition_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  log_date DATE NOT NULL,
  food_name VARCHAR(255),
  protein_g DECIMAL(6,2),
  carbs_g DECIMAL(6,2),
  fat_g DECIMAL(6,2),
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_nutrition_user_date 
  ON nutrition_logs(user_id, log_date);
```

---

## 3. API ENDPOINTS (RESTful)

### 3.1 Authentication (JWT, Phase 2)
```
POST   /api/auth/signup
POST   /api/auth/login
POST   /api/auth/refresh
GET    /api/auth/me           [Protected]
POST   /api/auth/logout       [Protected]
```

**Example: POST /api/auth/login**
```json
// Request
{
  "email": "user@athenetic.app",
  "password": "secure123"
}

// Response (200)
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 3600
}
```

---

### 3.2 Workout Logging

#### GET /api/sessions/:date
**Purpose**: Fetch all sets from a specific date  
**Auth**: Required (JWT)  
**Params**:
- `date`: ISO string (2026-05-08)

**Response**:
```json
{
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "session_date": "2026-05-08",
  "muscle_group": "CHEST",
  "readiness_score": 92,
  "sets": [
    {
      "id": "...",
      "exercise": { "id": "...", "name": "Barbell Bench Press", "muscle_group": "CHEST" },
      "weight_kg": 100,
      "reps_completed": 8,
      "rpe": 7,
      "rir": 2,
      "target_reps_next": 9,
      "created_at": "2026-05-08T14:30:00Z"
    }
  ]
}
```

#### POST /api/sessions
**Purpose**: Create new training session  
**Auth**: Required  
**Body**:
```json
{
  "session_date": "2026-05-08",
  "muscle_group": "CHEST",
  "readiness_score": 92
}

// Response (201)
{
  "session_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

#### POST /api/sets
**Purpose**: Log a single set with auto-calculated next target  
**Auth**: Required  
**Body**:
```json
{
  "session_id": "123e4567-e89b-12d3-a456-426614174000",
  "exercise_id": "abc123...",
  "weight_kg": 100.0,
  "reps_completed": 8,
  "rpe": 7,
  "rir": 2
}

// Response (201)
{
  "set_id": "def456...",
  "target_reps_next": 9,  // Auto-calculated
  "note": "Strong session; increment to 9 reps next time"
}
```

**Algorithm (Server-Side)**:
```go
func CalculateNextReps(pastRPE int, repsCompleted int, targetReps int) int {
  if pastRPE <= 6 && repsCompleted == targetReps {
    return targetReps + 2
  } else if (pastRPE >= 7 && pastRPE <= 8) && repsCompleted == targetReps {
    return targetReps + 1
  } else if pastRPE >= 9 || repsCompleted < targetReps {
    return targetReps  // Hold steady
  } else {
    return targetReps + 1  // Safe default
  }
}
```

#### GET /api/previous-set/:exercise_id
**Purpose**: Fetch previous session's data for pre-filling  
**Response**:
```json
{
  "weight_kg": 100.0,
  "reps_completed": 8,
  "rpe": 7,
  "target_reps_next": 9,
  "session_date": "2026-05-07",
  "notes": "Last session data"
}
```

---

### 3.3 Analytics & Volume Tracking

#### GET /api/analytics/volume
**Purpose**: Weekly volume aggregation by muscle group  
**Auth**: Required  
**Query Params**:
- `muscle_group`: CHEST | BACK | LEGS (optional; all if omitted)
- `weeks`: 1–52 (default: 1, current week)

**Response**:
```json
{
  "muscle_group": "CHEST",
  "week_start": "2026-05-05",
  "total_volume_load": 8400,      // Weight × Reps × Sets
  "total_sets": 12,
  "sessions": 2,
  "progress_vs_prior_week": 12.5  // % increase
}
```

**SQL Query** (GORM):
```go
var weeklyVolume struct {
  MuscleGroup string
  WeekStart   time.Time
  TotalVolume float64
  TotalSets   int
}

db.Model(&Set{}).
  Select("e.muscle_group, DATE_TRUNC('week', s.created_at) as week_start, SUM(s.weight_kg * s.reps_completed * 1) as total_volume, COUNT(s.id) as total_sets").
  Joins("JOIN exercises e ON sets.exercise_id = e.id").
  Joins("JOIN training_sessions ts ON sets.session_id = ts.id").
  Where("ts.user_id = ?", userID).
  Where("s.created_at >= NOW() - INTERVAL '1 week'").
  GroupBy("e.muscle_group, DATE_TRUNC('week', s.created_at)").
  Scan(&weeklyVolume)
```

#### GET /api/analytics/volume-progression
**Purpose**: 12-week historical trend (Phase 2)  
**Query Params**:
- `muscle_group`: CHEST | BACK | LEGS
- `weeks`: 1–52 (default: 12)

**Response**:
```json
[
  {
    "week": 1,
    "week_start": "2026-02-09",
    "volume_load": 7200,
    "target": 8000,
    "sets": 10,
    "rpe_avg": 7.2
  },
  {
    "week": 2,
    "week_start": "2026-02-16",
    "volume_load": 7800,
    "target": 8000,
    "sets": 11,
    "rpe_avg": 7.5
  },
  // ... weeks 3–12
]
```

#### GET /api/analytics/progress
**Purpose**: Compare week-to-week progress  
**Response**:
```json
{
  "current_week_load": 8400,
  "prior_week_load": 7200,
  "delta": 1200,
  "delta_percent": 16.7,
  "status": "Strong progress! 💪"
}
```

---

### 3.4 Readiness & Autoregulation (Phase 1)

#### POST /api/readiness
**Purpose**: Log daily readiness inputs  
**Auth**: Required  
**Body**:
```json
{
  "input_date": "2026-05-08",
  "sleep_hours": 7.5,
  "muscle_soreness": 4,        // 0–10 scale
  "running_km_prior_day": 5.0
}

// Response (201)
{
  "readiness_score": 78,       // Calculated
  "deload_factor": 0.85,       // Reduce 15%
  "deload_target_sets": 2,     // Default 3 → 2
  "recommendation": "Take it easy; reduce volume by 15%"
}
```

**Algorithm** (Server-Side):
```go
func CalculateReadinessScore(sleep float64, soreness int, runningKm float64) (score int, deloadFactor float64) {
  score = 100
  score -= int(math.Max(0, (8-sleep) * 5))
  score -= soreness * 2
  score -= int(runningKm * 1.5)
  
  score = max(50, min(100, score))  // Clamp to 50–100
  
  if score >= 85 {
    deloadFactor = 1.0
  } else if score >= 70 {
    deloadFactor = 0.85
  } else {
    deloadFactor = 0.70
  }
  
  return score, deloadFactor
}
```

#### GET /api/readiness/:date
**Purpose**: Fetch readiness data for specific date  
**Response**:
```json
{
  "input_date": "2026-05-08",
  "sleep_hours": 7.5,
  "muscle_soreness": 4,
  "running_km_prior_day": 5.0,
  "readiness_score": 78,
  "deload_factor": 0.85
}
```

---

### 3.5 Exercises (Catalog)

#### GET /api/exercises
**Purpose**: Fetch all exercises (or filter by muscle group)  
**Query Params**:
- `muscle_group`: CHEST | BACK | LEGS (optional)

**Response**:
```json
[
  {
    "id": "abc123...",
    "name": "Barbell Bench Press",
    "muscle_group": "CHEST",
    "default_target_sets": 3
  },
  {
    "id": "def456...",
    "name": "Dumbbell Flyes",
    "muscle_group": "CHEST",
    "default_target_sets": 3
  }
]
```

#### POST /api/exercises (Admin only, future)
**Purpose**: Add new exercise to catalog  
**Body**:
```json
{
  "name": "Archer Push-ups",
  "muscle_group": "CHEST",
  "default_target_sets": 3
}
```

---

### 3.6 Nutrition (Phase 3)

#### POST /api/nutrition/log
**Purpose**: Log food intake  
**Body**:
```json
{
  "log_date": "2026-05-08",
  "food_name": "Chicken Breast",
  "protein_g": 31.0,
  "carbs_g": 0.0,
  "fat_g": 3.6
}
```

#### GET /api/nutrition/summary/:date
**Purpose**: Daily totals  
**Response**:
```json
{
  "date": "2026-05-08",
  "body_weight_kg": 82,
  "protein_target": 130,  // 1.6g per kg
  "protein_logged": 95,
  "carbs_logged": 250,
  "fat_logged": 65,
  "total_calories": 1950
}
```

---

## 4. ERROR HANDLING

### 4.1 Standard Error Response
```json
{
  "error": true,
  "message": "Validation failed",
  "code": "VALIDATION_ERROR",
  "details": [
    { "field": "reps_completed", "issue": "must be positive integer" }
  ],
  "timestamp": "2026-05-08T14:30:00Z"
}
```

### 4.2 HTTP Status Codes
```
200 OK              – Successful GET / POST
201 Created         – Resource created
400 Bad Request     – Validation error
401 Unauthorized    – Missing/invalid JWT
403 Forbidden       – Not allowed
404 Not Found       – Resource doesn't exist
500 Internal Error  – Server error
```

---

## 5. MIDDLEWARE & SECURITY

### 5.1 CORS Configuration
```go
// Allow frontend requests
config := cors.DefaultConfig()
config.AllowOrigins = []string{
  "http://localhost:3000",         // Dev
  "https://athenetic.vercel.app",  // Prod
}
config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
```

### 5.2 JWT Middleware (Phase 2)
```go
func JWTMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
  return func(c echo.Context) error {
    auth := c.Request().Header.Get("Authorization")
    
    // Extract and validate token
    claims, err := ValidateJWT(auth)
    if err != nil {
      return echo.NewHTTPError(401, "Invalid token")
    }
    
    // Inject user_id into context
    c.Set("user_id", claims.UserID)
    return next(c)
  }
}
```

### 5.3 Input Validation
```go
// Always validate + sanitize inputs
func ValidateSetInput(input SetInput) error {
  if input.WeightKg <= 0 {
    return errors.New("weight must be positive")
  }
  if input.RepsCompleted < 1 || input.RepsCompleted > 100 {
    return errors.New("reps out of range")
  }
  if input.RPE < 1 || input.RPE > 10 {
    return errors.New("RPE must be 1–10")
  }
  return nil
}
```

### 5.4 Case Sensitivity Prevention
```go
// CRITICAL: Always use LOWER() for string comparisons
func (db *Database) GetExerciseByName(name string) (*Exercise, error) {
  var ex Exercise
  result := db.Where("LOWER(name) = LOWER(?)", name).First(&ex)
  return &ex, result.Error
}
```

---

## 6. DATABASE MIGRATIONS

### Approach: File-based SQL migrations (GORM auto-migrate)
```go
// internal/database/migrations.go

func RunMigrations(db *gorm.DB) error {
  return db.AutoMigrate(
    &User{},
    &Exercise{},
    &TrainingSession{},
    &Set{},
    &DailyReadinessInput{},
    &NutritionLog{},
  )
}
```

### Manual Migration (if needed)
```sql
-- migrations/001_create_exercises.sql
INSERT INTO exercises (name, muscle_group, default_target_sets) VALUES
  ('Barbell Bench Press', 'CHEST', 3),
  ('Dumbbell Flyes', 'CHEST', 3),
  ('Barbell Rows', 'BACK', 3),
  ('Pull-ups', 'BACK', 4),
  ('Leg Press', 'LEGS', 3),
  ('Deadlifts', 'LEGS', 3);
```

---

## 7. TESTING STRATEGY

### 7.1 Unit Tests (Algorithms)
```go
// internal/utils/calculations_test.go

func TestCalculateNextReps(t *testing.T) {
  tests := []struct {
    name              string
    pastRPE           int
    repsCompleted     int
    targetReps        int
    expectedNextReps  int
  }{
    {"RPE 6, completed target", 6, 8, 8, 10},
    {"RPE 7, completed target", 7, 8, 8, 9},
    {"RPE 9, hold steady", 9, 7, 8, 8},
    {"Failed reps, hold steady", 7, 6, 8, 8},
  }
  
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      result := CalculateNextReps(tt.pastRPE, tt.repsCompleted, tt.targetReps)
      if result != tt.expectedNextReps {
        t.Errorf("got %d, want %d", result, tt.expectedNextReps)
      }
    })
  }
}
```

### 7.2 Integration Tests (API Endpoints)
```go
// handlers/workout_test.go

func TestPostSet(t *testing.T) {
  // Setup: Create session, exercise
  // Call: POST /api/sets with valid data
  // Assert: Response 201, target_reps_next correct, DB updated
}
```

### 7.3 Load Testing (Future)
```bash
# k6 load test
k6 run load-test.js
# Target: 1000 concurrent users, <200ms p99 latency
```

---

## 8. DEPLOYMENT

### 8.1 Docker Compose (Local)
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: athenetic
      POSTGRES_USER: admin
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  backend:
    build: .
    environment:
      DATABASE_URL: postgres://admin:password@postgres:5432/athenetic
      JWT_SECRET: dev-secret-key
      PORT: 8080
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    command: /app/server

volumes:
  postgres_data:
```

### 8.2 Environment Variables
```
DATABASE_URL=postgres://user:pass@host:5432/db
JWT_SECRET=<32-char-random-key>
JWT_EXPIRY_HOURS=1
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://athenetic.vercel.app
LOG_LEVEL=info
PORT=8080
```

### 8.3 Production Deployment
```bash
# Build
docker build -t athenetic-backend:latest .

# Run on Cloud Run, ECS, or Heroku
docker run \
  -e DATABASE_URL=prod-db-url \
  -e JWT_SECRET=prod-secret \
  -p 8080:8080 \
  athenetic-backend:latest
```

---

## 9. PERFORMANCE OPTIMIZATION

### 9.1 Database Indexing
```sql
CREATE INDEX idx_sets_exercise_date ON sets(exercise_id, created_at DESC);
CREATE INDEX idx_sessions_user_date ON training_sessions(user_id, session_date DESC);
CREATE INDEX idx_nutrition_user_date ON nutrition_logs(user_id, log_date DESC);

-- For volume aggregations
CREATE INDEX idx_sets_created_at ON sets(created_at DESC);
```

### 9.2 Query Optimization
```go
// ✓ Correct: Single query with JOIN + aggregation
db.Model(&Set{}).
  Select("e.muscle_group, SUM(s.weight * s.reps) as volume").
  Joins("JOIN exercises e ON sets.exercise_id = e.id").
  GroupBy("e.muscle_group").
  Scan(&results)

// ✗ Wrong: N+1 query problem
for _, set := range sets {
  exercise := db.First(&Exercise{}, set.ExerciseID) // Repeated queries!
}
```

### 9.3 Caching Strategy (Future)
```go
// Cache volume aggregations for 1 hour
cache.Set("volume:CHEST:2026-05", volumeData, 1*time.Hour)
```

---

## 10. ROADMAP: Backend Implementation Phases

| Phase | Duration | Features |
|-------|----------|----------|
| **1** | 2–3 wks  | Readiness scoring, Exercise expansion, Route restructuring |
| **2** | 4–6 wks  | JWT auth, User accounts, 12-week analytics endpoints |
| **3** | 8–12 wks | Nutrition tracking, Caching optimization |

---

**Reference**: [PRODUCT_SPECIFICATION.md](../PRODUCT_SPECIFICATION.md)  
**Last Updated**: May 8, 2026
