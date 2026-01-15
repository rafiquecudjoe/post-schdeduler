# Social Media Post Scheduler

A minimal Buffer-like social media post scheduler built with Go, Next.js, PostgreSQL, and Redis.

## 🔒 Security Notice

**⚠️ IMPORTANT: This application requires proper environment configuration before running.**

The application enforces secure configuration and will **not start** without:
- A cryptographically secure JWT secret (32+ characters)
- Proper database connection string
- Redis connection URL

**Quick Security Setup:**
```bash
# 1. Copy environment template
cp .env.example .env.development

# 2. Generate a secure JWT secret
openssl rand -base64 48

# 3. Update .env.development with your generated secret

# 4. Review and update other required variables
```

📖 **See [ENV_SETUP.md](ENV_SETUP.md) for complete configuration guide**

⚠️ **Never commit `.env*` files with real secrets to version control!**

## 🏗️ Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Next.js App   │────▶│   Go API Server │────▶│   PostgreSQL    │
│   (Frontend)    │     │   (Backend)     │     │   (Database)    │
└─────────────────┘     └────────┬────────┘     └─────────────────┘
                                 │
                                 ▼
                        ┌─────────────────┐
                        │     Redis       │◀───┐
                        │ (Queue/Cache)   │    │
                        └────────┬────────┘    │
                                 │             │
                                 ▼             │
                        ┌─────────────────┐    │
                        │   Go Worker     │────┘
                        │ (Background)    │
                        └─────────────────┘
```

## ✨ Features

### Core Functionality
- **User Authentication**: JWT-based auth with secure HTTP-only cookies
- **Post Scheduling**: Create, edit, and delete scheduled posts
- **Multi-Channel Support**: Twitter, LinkedIn, Facebook channels
- **Background Publishing**: Reliable queue-based publishing with Redis
- **Real-time Updates**: Server-Sent Events (SSE) for live post status
- **Dashboard**: View upcoming scheduled posts and publishing history

### Security Features 🔒
- **Strong Password Policy**: 12+ chars, uppercase, lowercase, digits, special characters
- **RFC-Compliant Email Validation**: Proper email format validation
- **Rate Limiting**: Multi-tier rate limiting per endpoint
  - Auth endpoints: 5 req/min
  - Registration: 3 req/min
  - Post creation: 30 req/min
  - General API: 100 req/min
- **Secure Cookies**: HttpOnly, Secure flag (HTTPS), SameSite Strict
- **Token Blacklisting**: Revoked tokens stored in Redis
- **SQL Injection Protection**: Parameterized queries throughout
- **CORS Configuration**: Strict origin validation
- **Input Validation**: Content length limits, sanitization
- **Fail-Secure Rate Limiting**: Service fails closed if Redis unavailable

### Performance & Reliability
- **Caching**: Redis caching for post lists with automatic invalidation
- **Background Jobs**: Asynchronous post publishing with retry logic
- **Database Migrations**: Automated schema management
- **Health Checks**: Service health monitoring

## 🛠️ Tech Stack

| Component | Technology |
|-----------|------------|
| Backend API | Go 1.22 with Chi router |
| Frontend | Next.js 14 (App Router) |
| Database | PostgreSQL 15 |
| Queue/Cache | Redis 7 |
| Containerization | Docker & Docker Compose |
| CI/CD | GitHub Actions |

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Git

### Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd post-scheduler
   ```

2. **Configure environment variables**
   ```bash
   # Copy the example environment file
   cp .env.example .env.development
   
   # Generate a secure JWT secret (32+ characters required)
   openssl rand -base64 48
   
   # Edit .env.development and:
   # - Replace JWT_SECRET with generated value
   # - Update DATABASE_URL if needed
   # - Set other variables as needed
   ```

3. **Start all services**
   ```bash
   docker-compose up --build
   ```

3. **Access the application**
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080

4. **Create a test user**
   ```bash
   curl -X POST http://localhost:8080/api/auth/register \
     -H "Content-Type: application/json" \
     -d '{"email": "test@example.com", "password": "password123"}'
   ```

## 📦 Project Structure

```
post-scheduler/
├── backend/
│   ├── cmd/server/          # Application entrypoint
│   ├── internal/
│   │   ├── api/             # HTTP handlers & middleware
│   │   ├── auth/            # JWT & password handling
│   │   ├── cache/           # Redis caching layer
│   │   ├── db/              # Database migrations & queries
│   │   ├── models/          # Data models & validation
│   │   ├── scheduler/       # Redis queue & worker
│   │   └── config/          # Environment configuration
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── app/                 # Next.js App Router pages
│   ├── components/          # React components
│   ├── lib/                 # Utilities & API client
│   ├── Dockerfile
│   └── package.json
├── docker-compose.yml
└── README.md
```

## 🔄 How Scheduling Works

1. **User creates a post** with a future `scheduled_at` timestamp
2. **Backend enqueues** the post ID in a Redis sorted set (score = Unix timestamp)
3. **Worker polls** Redis every 10 seconds for posts where `scheduled_at <= now`
4. **Worker publishes** the post: updates status to "published", sets `published_at`
5. **Post moves** from "Upcoming" to "History" in the dashboard

### Demo: Testing the Publishing Flow

1. Create a post scheduled 1 minute in the future
2. Watch the worker logs: `docker-compose logs -f worker`
3. After 1 minute, see the "📤 Published post" message
4. Refresh the dashboard to see the post in History

## 🔐 Authentication

- **JWT Access Token**: 15-minute TTL, stored in HTTP-only cookie
- **JWT Refresh Token**: 7-day TTL, stored in HTTP-only cookie
- **Token Refresh**: Automatic via `/api/auth/refresh` endpoint
- **Logout**: Tokens blacklisted in Redis

## 📝 API Endpoints

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Create new user |
| POST | `/api/auth/login` | Login, set JWT cookies |
| POST | `/api/auth/logout` | Blacklist tokens |
| POST | `/api/auth/refresh` | Refresh access token |
| GET | `/api/auth/me` | Get current user |

### Posts
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/posts` | Create scheduled post |
| GET | `/api/posts/upcoming` | List scheduled posts |
| GET | `/api/posts/history` | List published posts |
| GET | `/api/posts/:id` | Get single post |
| PUT | `/api/posts/:id` | Update scheduled post |
| DELETE | `/api/posts/:id` | Delete scheduled post |

## 🧪 Running Tests

```bash
# Backend unit tests
cd backend && go test -v ./...

# Cypress E2E tests (headless)
cd frontend && npm run test:e2e

# Cypress E2E tests (interactive)
cd frontend && npm run cypress
```

### E2E Test Suites

| Suite | Tests |
|-------|-------|
| `auth.cy.ts` | Registration, login, logout, protected routes |
| `posts.cy.ts` | Create, edit, delete posts, view tabs |
| `full-flow.cy.ts` | Complete scheduling → publishing flow |

## 🏃 Running the Worker Separately

The worker runs as a separate container by default. To run manually:

```bash
# In Docker
docker-compose exec backend ./server --worker

# Locally (requires Go)
cd backend && go run ./cmd/server --worker
```

## 📊 Architecture Decisions

### Why Redis Sorted Sets for Scheduling?

- **Time-based ordering**: `ZRANGEBYSCORE` efficiently retrieves due posts
- **Atomic operations**: `ZREM` prevents duplicate processing
- **Simple**: No external message broker needed

### Why JWT in HTTP-only Cookies?

- **XSS protection**: JavaScript cannot access the tokens
- **Stateless**: No session storage required (except blacklist)
- **Scalable**: Works across multiple backend instances

### Why sqlc over GORM?

- **Type safety**: Compile-time SQL validation
- **Performance**: Raw SQL queries
- **Explicit**: Full control over database operations

## 🎁 Bonus Features Implemented

### Rate Limiting
- Redis-based sliding window rate limiting
- Configurable per-endpoint limits:
  - Login: 5 requests/minute
  - Registration: 3 requests/minute
  - Post creation: 30 requests/minute
  - General API: 100 requests/minute
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

### Redis Caching
- Cached endpoints: `/api/posts/upcoming` (30s TTL), `/api/posts/history` (60s TTL)
- Automatic cache invalidation on create/update/delete
- Cache-aside pattern with fail-open behavior

### Edit Updates Queue
- When editing a post's `scheduled_at`, the Redis queue is updated atomically

### Real-time Updates (SSE)
- **Server-Sent Events** endpoint: `GET /api/posts/stream`
- Pushes updates every 10 seconds when data changes
- Auto-reconnect on connection loss
- React hook: `usePostStream()` for easy integration
- Zero external dependencies (uses Go stdlib + browser EventSource API)

### Error States & Retry Mechanism
- Failed posts are marked with `status: "failed"` and `last_error` message
- **Exponential backoff retry**: Up to 3 retries with delays of 2, 4, 8 minutes
- Worker handles errors gracefully without crashing
- Retry tracking: `retry_count`, `last_error`, `next_retry_at` fields

## ⚠️ Known Limitations

1. **No real social media integration** - Publishing is mocked
2. **No media uploads** - Text content only
3. **Single timezone** - All times stored in UTC, displayed in browser's local time

## 🔮 Future Improvements

- [ ] Media upload support (images, videos)
- [ ] User timezone preferences
- [ ] Team/organization features
- [ ] Analytics dashboard
- [x] ~~Rate limiting~~ ✅
- [x] ~~Redis caching~~ ✅
- [x] ~~Retry mechanism with exponential backoff~~ ✅
- [x] ~~Cypress E2E tests~~ ✅
- [x] ~~Real-time updates via SSE~~ ✅

## 📹 Demo Video

**🎬 [Watch the Full Demo Video](YOUR_VIDEO_LINK_HERE)**

*10-minute walkthrough covering:*
- Local setup with Docker Compose
- User registration and authentication
- Creating and scheduling posts
- Background worker publishing posts in real-time
- Real-time SSE updates without page refresh
- Code architecture highlights
- Security features and rate limiting

*Video will demonstrate the complete scheduling → publishing flow with a post scheduled 1 minute in the future, showing live worker logs as it processes and publishes the post.*

## 📄 License

MIT
