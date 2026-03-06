# 🎵 Playlist Tracker

A web app that allows users to **transfer and sync playlists** across multiple music streaming platforms. Built with a Next.js frontend, Go backend, PostgreSQL, and Docker-based local deployment.

[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![Next.js](https://img.shields.io/badge/Next.js-16-000000?logo=next.js&logoColor=white)](https://nextjs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)

---

## ✨ Features

### 🔐 Authentication & Security
- **Google OAuth2** for secure user authentication
- **JWT-based** session management with automatic token refresh
- Token expiry monitoring and health checks
- Secure token revocation on service disconnect

### 🎧 Music Platform Integration
- **Spotify** - Full playlist read/write access
- **YouTube Music** - Playlist management and video integration
- Automatic OAuth token refresh before expiry
- Service connection health monitoring

### 🔄 Smart Playlist Transfer
- **Intelligent track matching** with confidence scoring (0.0 - 1.0)
- Cross-platform track search with fuzzy matching
- YouTube title parsing to extract artist and track information
- Detailed transfer history with per-track status
- Async background processing for large playlists
- Handles incomplete matches gracefully

### 🚦 Enterprise Rate Limiting
- **Token bucket algorithm** for API rate limiting
- Service-specific limits (Spotify: 10 req/s, YouTube: 1 req/s)
- Automatic retry with exponential backoff
- Real-time rate limit monitoring and metrics
- Prevents API quota exhaustion

### 📊 Transfer Tracking
- Automatic transfer status polling in the dashboard
- Track-level success/failure reporting
- Match confidence scores for each track
- Failed track identification for manual review
- Complete transfer history with timestamps

---

## 🏗️ Architecture

### Tech Stack

| Layer | Technology | Version |
|-------|------------|---------|
| **Frontend** | Next.js | 16.0.1 |
| | React | 19.2.0 |
| | TypeScript | 5.x |
| | Tailwind CSS | 4.x |
| | Axios | 1.13.2 |
| **Backend** | Go | 1.24.0 |
| | Gin Web Framework | Latest |
| | GORM | Latest |
| | OAuth2 | golang.org/x/oauth2 |
| **Database** | PostgreSQL | 15-alpine |
| **Infrastructure** | Docker & Docker Compose | Latest |
| | Nginx | alpine |
| **Authentication** | Google OAuth2 + JWT | - |

### System Design

```
┌─────────────────────────────────────────────────────────────┐
│                         Nginx (Port 80)                     |
│                    Reverse Proxy & Router                   │
└─────────────────────┬───────────────────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        │                           │
        ▼                           ▼
┌───────────────┐           ┌───────────────┐
│  Next.js      │           │   Go Server   │
│  Client       │◄─────────►│   (Gin)       │
│  Port 3000    │   REST    │   Port 8080   │
└───────────────┘           └───────┬───────┘
                                    │
                            ┌───────┴────────┐
                            │                │
                            ▼                ▼
                    ┌──────────────┐  ┌──────────────┐
                    │  PostgreSQL  │  │  External    │
                    │  Port 5432   │  │  APIs        │
                    └──────────────┘  │ - Spotify    │
                                      │ - YouTube    │
                                      └──────────────┘
```

---

## 📁 Project Structure

```
playlist-tracker/
├── client/                          # Next.js Frontend
│   ├── app/
│   │   ├── auth/
│   │   │   └── success/
│   │   │       └── page.tsx         # OAuth callback handler
│   │   ├── components/
│   │   │   ├── Playlists.tsx        # Playlist display component
│   │   │   ├── TransferHistory.tsx  # Transfer tracking UI
│   │   │   └── TransferModal.tsx    # Transfer initiation modal
│   │   ├── dashboard/
│   │   │   └── page.tsx             # Main dashboard
│   │   └── page.tsx                 # Landing page
│   ├── dockerfile
│   └── package.json
│
├── server/                          # Go Backend
│   ├── internal/
│   │   ├── auth/
│   │   │   ├── oauth.go            # OAuth configurations
│   │   │   └── token_manager.go    # Token lifecycle management
│   │   ├── database/
│   │   │   └── database.go         # Models & DB connection
│   │   ├── handlers/
│   │   │   ├── auth.go             # Authentication handlers
│   │   │   ├── playlists.go        # Playlist operations
│   │   │   ├── services.go         # Service connections
│   │   │   └── transfers.go        # Transfer processing
│   │   ├── middleware/
│   │   │   └── auth.go             # JWT middleware
│   │   └── ratelimit/
│   │       ├── rate_limiter.go     # Token bucket implementation
│   │       ├── http_client.go      # Rate-limited HTTP client
│   │       └── monitor.go          # Metrics & monitoring
│   ├── main.go
│   ├── dockerfile
│   ├── go.mod
│   └── go.sum
│
├── nginx/
│   └── nginx.conf                   # Reverse proxy configuration
│
├── docker-compose.yml               # Multi-container orchestration
├── .env.example                     # Environment template
└── README.md
```

---

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Google Cloud Console account (for OAuth credentials)
- Spotify Developer account
- YouTube Data API v3 credentials

### 1. Clone Repository

```bash
git clone https://github.com/yourusername/playlist-tracker.git
cd playlist-tracker
```

### 2. Configure Environment Variables

Create a `.env` file in the root directory:

```bash
cp .env.example .env
```

Edit `.env` with your credentials:

```env
# Google OAuth (for app login)
GOOGLE_OAUTH_CLIENT_ID=your-google-client-id
GOOGLE_OAUTH_CLIENT_SECRET=your-google-client-secret

# Spotify API
SPOTIFY_CLIENT_ID=your-spotify-client-id
SPOTIFY_CLIENT_SECRET=your-spotify-client-secret

# YouTube Data API v3
YOUTUBE_CLIENT_ID=your-youtube-client-id
YOUTUBE_CLIENT_SECRET=your-youtube-client-secret

# JWT Secret (generate a strong random string)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Token Encryption Key (32 raw characters or base64-encoded 32-byte key)
TOKEN_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef

# URLs (adjust for production)
FRONTEND_URL=http://localhost:3000
BACKEND_URL=http://localhost:8080

# Rate Limiting Configuration
SPOTIFY_REQUESTS_PER_SECOND=10
SPOTIFY_BURST_LIMIT=20
YOUTUBE_REQUESTS_PER_SECOND=1
YOUTUBE_BURST_LIMIT=5
```

### 3. OAuth Setup

#### Google OAuth (App Login)
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project
3. Enable Google+ API
4. Create OAuth 2.0 credentials
5. Add authorized redirect URI: `http://localhost:8080/api/auth/google/callback`

#### Spotify
1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Create an app
3. Add redirect URI: `http://localhost:8080/api/services/callback/spotify`

#### YouTube
1. Use the same Google Cloud project
2. Enable YouTube Data API v3
3. Add redirect URI: `http://localhost:8080/api/services/callback/youtube`

### 4. Launch Application

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Check status
docker-compose ps
```

### 5. Access Application

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **Nginx**: http://localhost:80

---

## 🔧 Development

### Running Without Docker

#### Frontend (Next.js)

```bash
cd client
npm install
npm run dev
```

#### Backend (Go)

```bash
cd server
go mod download
go run main.go
```

#### Database

```bash
# Start PostgreSQL
docker run -d \
  --name postgres \
  -e POSTGRES_DB=playlist_tracker \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  postgres:15-alpine
```

### Hot Reload

- **Frontend**: Hot reload is enabled by default with Next.js dev server
- **Backend**: Consider using [Air](https://github.com/cosmtrek/air) for Go hot reload

```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Run with hot reload
cd server
air
```

---

## 📡 API Reference

### Authentication Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/api/auth/google` | GET | Initiate Google OAuth flow | No |
| `/api/auth/google/callback` | GET | OAuth callback handler | No |
| `/api/auth/me` | GET | Get current user info | Yes |
| `/api/auth/logout` | POST | Logout user | Yes |

### Service Connection Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/api/services` | GET | Get connected services | Yes |
| `/api/services/connect/:provider` | POST | Create an authenticated Spotify/YouTube OAuth URL | Yes |
| `/api/services/callback/:provider` | GET | Service OAuth callback | No |
| `/api/services/:provider` | DELETE | Disconnect service | Yes |
| `/api/services/health` | GET | Token health check | Yes |

### Playlist Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/api/playlists/:service` | GET | Fetch playlists from service | Yes |
| `/api/playlists/:service/stored` | GET | Get cached playlists | Yes |
| `/api/playlists/sync` | POST | Sync all playlists | Yes |

### Transfer Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/api/transfers` | POST | Start playlist transfer | Yes |
| `/api/transfers` | GET | Get transfer history | Yes |
| `/api/transfers/:id` | GET | Get transfer details | Yes |

### Monitoring Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/api/rate-limits` | GET | Get rate limit stats | Yes |
| `/api/health` | GET | Health check | No |

---

## 🎯 How It Works

### Transfer Flow

1. **User Initiates Transfer**
   - Select source service and playlist
   - Select target service
   - Optionally rename target playlist

2. **Backend Processing**
   - Validates service connections
   - Creates transfer record in database
   - Starts async goroutine for processing

3. **Track Matching**
   - Fetches all tracks from source playlist
   - For each track:
     - Searches on target service
     - Calculates match confidence
     - Records match details

4. **Playlist Creation**
   - Creates new playlist on target service
   - Adds matched tracks
   - Skips unmatched tracks

5. **Result Recording**
   - Updates transfer status
   - Records per-track results
   - Calculates success metrics

### Track Matching Algorithm

```go
// Confidence calculation (0.0 - 1.0)
confidence = 0.0

// Exact name match: +0.6
if sourceTrackName == targetTrackName {
    confidence += 0.6
}

// Exact artist match: +0.4
if sourceArtist == targetArtist {
    confidence += 0.4
}

// Partial matches: +0.2-0.4
// Total: 0.0 (no match) to 1.0 (perfect match)
```

### Rate Limiting

Uses **token bucket algorithm**:
- Spotify: 10 requests/second, burst up to 20
- YouTube: 1 request/second, burst up to 5
- Automatic backoff on 429 responses
- Retry with exponential delay

---

## 🗄️ Database Schema

### Users
```sql
id, google_id, email, name, avatar_url, created_at, updated_at
```

### UserServices
```sql
id, user_id, service_type, access_token, refresh_token, 
token_expiry, service_user_id, service_user_name, created_at, updated_at
```

### Playlists
```sql
id, user_id, service_type, service_id, name, description, 
track_count, image_url, is_public, last_synced_at, created_at, updated_at
```

### Transfers
```sql
id, user_id, source_service, source_playlist_id, source_playlist_name,
target_service, target_playlist_id, target_playlist_name, status,
tracks_total, tracks_matched, tracks_failed, error_message, created_at, updated_at
```

### TransferTracks
```sql
id, transfer_id, source_track_id, source_track_name, source_artist,
target_track_id, target_track_name, target_artist, status, 
match_confidence, created_at, updated_at
```

---

## 🐳 Docker Configuration

### Services

- **postgres**: PostgreSQL 15 database
- **server**: Go backend API
- **client**: Next.js frontend
- **nginx**: Reverse proxy

### Volumes

- `postgres_data`: Persistent database storage

### Networks

- `app-network`: Bridge network for inter-service communication

### Building Images

```bash
# Build all services
docker-compose build

# Build specific service
docker-compose build server

# Rebuild without cache
docker-compose build --no-cache
```

### Managing Containers

```bash
# Start services
docker-compose up -d

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v

# View logs
docker-compose logs -f [service_name]

# Restart service
docker-compose restart [service_name]
```

---

## 🔒 Security Considerations

### Production Checklist

- [ ] Change JWT_SECRET to a strong random value
- [ ] Use HTTPS for all endpoints
- [ ] Implement rate limiting at nginx level
- [ ] Add CORS restrictions for production domain
- [ ] Use Docker secrets instead of environment variables
- [ ] Enable PostgreSQL SSL connections
- [ ] Implement refresh token rotation
- [ ] Add request logging and monitoring
- [ ] Set up automated backups for PostgreSQL
- [ ] Use non-root users in Docker containers

### Environment Security

```bash
# Generate secure JWT secret
openssl rand -base64 64

# Generate a 32-byte token encryption key
openssl rand -base64 32
```

---

## 📊 Monitoring & Metrics

### Available Metrics

Access at `/api/rate-limits`:

```json
{
  "rate_limits": {
    "spotify": {
      "total_requests": 1250,
      "rate_limited": 3,
      "errors": 5,
      "last_request_time": "2024-01-15T10:30:00Z"
    },
    "youtube": {
      "total_requests": 450,
      "rate_limited": 1,
      "errors": 2,
      "last_request_time": "2024-01-15T10:29:45Z"
    }
  }
}
```

### Health Checks

```bash
# Application health
curl http://localhost:8080/api/health

# Protected endpoints use the session cookie established after browser login.
# Example with an exported cookie jar:
curl -b cookies.txt http://localhost:8080/api/services/health
```

---

## 🧪 Testing

### Manual Testing

```bash
# Test authentication
curl http://localhost:8080/api/health

# After logging in via the browser, export/import the session cookie and use:
curl -b cookies.txt http://localhost:8080/api/auth/me
```

### Database Access

```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U user -d playlist_tracker

# View transfers
SELECT * FROM transfers ORDER BY created_at DESC LIMIT 10;

# View transfer tracks
SELECT * FROM transfer_tracks WHERE transfer_id = 1;
```

---

## 🚧 Troubleshooting

### Common Issues

**Port Already in Use**
```bash
# Check port usage
lsof -i :3000  # or :8080, :5432

# Stop conflicting service or change port in docker-compose.yml
```

**Database Connection Failed**
```bash
# Verify PostgreSQL is running
docker-compose ps postgres

# Check logs
docker-compose logs postgres

# Restart database
docker-compose restart postgres
```

**OAuth Redirect URI Mismatch**
- Ensure redirect URIs in OAuth provider match exactly
- Check for http vs https
- Verify port numbers

**Rate Limit Errors**
- Check `/api/rate-limits` for current status
- Wait for rate limit reset
- Consider reducing concurrent requests

---

## 🗺️ Roadmap

### Phase 1: Core Features ✅
- [x] Google OAuth authentication
- [x] Spotify integration
- [x] YouTube Music integration
- [x] Basic playlist transfer
- [x] Transfer history

### Phase 2: Enhanced Features ✅
- [x] Rate limiting system
- [x] Token refresh management
- [x] Match confidence scoring
- [x] Docker containerization

### Phase 3: Future Enhancements
- [ ] Apple Music support
- [ ] Amazon Music support
- [ ] Scheduled automatic syncs
- [ ] Playlist diff/merge functionality
- [ ] Collaborative playlist management
- [ ] Email notifications for transfers
- [ ] Mobile app (React Native)
- [ ] Advanced fuzzy matching with ML
- [ ] Playlist analytics dashboard

---

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit changes (`git commit -m 'Add AmazingFeature'`)
4. Push to branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Code Style

- **Go**: Follow [Effective Go](https://golang.org/doc/effective_go.html)
- **TypeScript/React**: Use ESLint configuration provided
- **Commits**: Use conventional commits format

---

## 🙏 Acknowledgments

- [Spotify Web API](https://developer.spotify.com/documentation/web-api)
- [YouTube Data API v3](https://developers.google.com/youtube/v3)
- [Gin Web Framework](https://gin-gonic.com/)
- [Next.js](https://nextjs.org/)
- [GORM](https://gorm.io/)

---

## 📞 Support

For issues, questions, or suggestions:
- Open an [Issue](https://github.com/yourusername/playlist-tracker/issues)
- Start a [Discussion](https://github.com/yourusername/playlist-tracker/discussions)

---

**Made with ❤️ for music lovers everywhere**
