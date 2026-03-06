"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { API_BASE_URL, api } from "../lib/api";

interface User {
  id: number;
  email: string;
  name: string;
  avatarURL?: string;
}

export default function Home() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    checkAuth();
  }, []);

  const checkAuth = async () => {
    try {
      const response = await api.get("/api/auth/me");
      setUser(response.data.user);
    } catch {
      setUser(null);
    } finally {
      setLoading(false);
    }
  };

  const handleGoogleLogin = () => {
    window.location.href = `${API_BASE_URL}/api/auth/google`;
  };

  const handleLogout = async () => {
    try {
      await api.post("/api/auth/logout");
    } catch (error) {
      console.error("Logout error:", error);
    } finally {
      setUser(null);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md mx-auto bg-white rounded-lg shadow-md p-6">
        <h1 className="text-3xl font-bold text-center text-gray-900 mb-8">
          Playlist Tracker
        </h1>

        {user ? (
          <div className="text-center">
            <p className="text-lg mb-4 text-black">Welcome, {user.name}!</p>
            <div className="space-y-4">
              <button
                onClick={() => router.push('/dashboard')}
                className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700"
              >
                Go to Dashboard
              </button>
              <button
                onClick={handleLogout}
                className="w-full bg-gray-600 text-white py-2 px-4 rounded-md hover:bg-gray-700"
              >
                Logout
              </button>
            </div>
          </div>
        ) : (
          <div className="text-center">
            <p className="text-lg text-black mb-6">Transfer playlists between music services</p>
            <button
              onClick={handleGoogleLogin}
              className="w-full bg-red-600 text-white py-2 px-4 rounded-md hover:bg-red-700 flex items-center justify-center"
            >
              <span>Login with Google</span>
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
