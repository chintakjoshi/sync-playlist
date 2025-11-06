# Playlist Tracker

Playlist Tracker is a Progressive Web App (PWA) that allows users to **transfer and sync playlists** across multiple music platforms including **YouTube Music, Spotify, Apple Music, and Amazon Music**.  
It eliminates the pain of manually recreating playlists when switching between services.

---

## 🚀 Features

- Login via Google OAuth2  
- Connect to Spotify and YouTube Music accounts  
- Fetch, display, and sync playlists across platforms  
- Start and monitor playlist transfers with progress and error tracking  
- Token-based authentication using JWT  
- Transfer history with per-track match confidence  
- Built as a **Next.js (React)** client and **Go (Gin + GORM)** backend

---

## 🧩 Tech Stack

| Layer | Technology |
|-------|-------------|
| Frontend | Next.js 16, React 19, TypeScript, Tailwind CSS 4 |
| Backend | Go 1.22+, Gin, GORM, PostgreSQL |
| Auth | Google OAuth2 + JWT |
| Deployment | Docker (optional) |

---

## ⚙️ Project Structure

```
/client
 ├── app/
 │   ├── auth/
 │   ├── components/
 │   ├── dashboard/
 │   └── page.tsx
 ├── package.json
 └── tailwind.config.js

/server
 ├── main.go
 ├── routes/
 ├── models/
 ├── handlers/
 ├── utils/
 └── config/
```

---

## 🛠️ Setup Instructions

### 1. Clone Repository

```bash
git clone https://github.com/yourusername/playlist-tracker.git
cd playlist-tracker
```

### 2. Environment Variables

Create a `.env` file in both `client/` and `server/` directories.

#### Example `.env` for Backend

```
PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=playlisttracker

JWT_SECRET=your_jwt_secret
FRONTEND_URL=http://localhost:3000

GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
SPOTIFY_CLIENT_ID=your_spotify_client_id
SPOTIFY_CLIENT_SECRET=your_spotify_client_secret
YOUTUBE_API_KEY=your_youtube_api_key
```

#### Example `.env.local` for Frontend

```
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### 3. Install Dependencies

#### Frontend

```bash
cd client
npm install
npm run dev
```

#### Backend

```bash
cd server
go mod tidy
go run main.go
```

---

## 🧠 API Overview

| Endpoint | Method | Description |
|-----------|---------|-------------|
| `/api/auth/google` | GET | Google OAuth login |
| `/api/auth/me` | GET | Get user info |
| `/api/services/connect/:provider` | GET | Connect Spotify/YouTube |
| `/api/playlists/:service` | GET | Fetch playlists |
| `/api/transfers` | POST | Start transfer |
| `/api/transfers/:id` | GET | Get transfer details |

---

## 🧪 Development Notes

- Ensure PostgreSQL is running before backend startup  
- Use `localhost:8080` for API and `localhost:3000` for frontend during development  
- Tokens are stored in `localStorage` on the client  
- Background goroutines handle playlist transfers asynchronously

---

## 📦 Deployment

You can use Docker Compose or deploy frontend and backend separately. Example Dockerfile provided in `/server`.

---

## 🧰 Future Improvements

- Token encryption at rest  
- Refresh token rotation  
- Pagination support for large playlists  
- Support for Apple Music and Amazon Music  
- Improved fuzzy matching algorithm

---