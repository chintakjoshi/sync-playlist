"use client";

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import axios from 'axios';

interface User {
    id: number;
    email: string;
    name: string;
    avatarURL?: string;
}

export default function Dashboard() {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);
    const [message, setMessage] = useState<string>('');
    const router = useRouter();
    const searchParams = useSearchParams();

    useEffect(() => {
        checkAuth();

        // Check for success messages from OAuth redirects
        const messageParam = searchParams.get('message');
        if (messageParam) {
            if (messageParam === 'spotify_connected') {
                setMessage('Successfully connected to Spotify!');
            } else if (messageParam === 'youtube_connected') {
                setMessage('Successfully connected to YouTube Music!');
            }

            // Clear the message after 5 seconds
            setTimeout(() => setMessage(''), 5000);
        }
    }, [searchParams]);

    const checkAuth = async () => {
        try {
            const token = localStorage.getItem('token');
            console.log('Dashboard token check:', token);

            if (!token) {
                console.log('No token, redirecting to home');
                router.push('/');
                return;
            }

            const response = await axios.get('http://localhost:8080/api/auth/me', {
                headers: { Authorization: `Bearer ${token}` }
            });
            setUser(response.data.user);
            console.log('User data:', response.data.user);
        } catch (error) {
            console.error('Auth check failed:', error);
            localStorage.removeItem('token');
            router.push('/');
        } finally {
            setLoading(false);
        }
    };

    const handleConnectService = (provider: string) => {
        window.location.href = `http://127.0.0.1:8080/api/services/connect/${provider}`;
    };

    const handleLogout = async () => {
        try {
            const token = localStorage.getItem('token');
            if (token) {
                await axios.post('http://localhost:8080/api/auth/logout', {}, {
                    headers: { Authorization: `Bearer ${token}` }
                });
            }
        } catch (error) {
            console.error('Logout error:', error);
        } finally {
            localStorage.removeItem('token');
            router.push('/');
        }
    };

    if (loading) {
        return (
            <div className="min-h-screen flex items-center justify-center">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50">
            {/* Header */}
            <header className="bg-white shadow">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex justify-between items-center">
                    <h1 className="text-2xl font-bold text-gray-900">Playlist Tracker</h1>
                    <div className="flex items-center space-x-4">
                        {user && (
                            <div className="flex items-center space-x-2">
                                {user.avatarURL && (
                                    <img src={user.avatarURL} alt={user.name} className="w-8 h-8 rounded-full" />
                                )}
                                <span className="text-gray-700">Welcome, {user.name}</span>
                            </div>
                        )}
                        <button
                            onClick={handleLogout}
                            className="bg-gray-600 text-white py-2 px-4 rounded-md hover:bg-gray-700"
                        >
                            Logout
                        </button>
                    </div>
                </div>
            </header>

            {/* Success Message */}
            {message && (
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-4">
                    <div className="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded">
                        {message}
                    </div>
                </div>
            )}

            {/* Main Content */}
            <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    {/* Service Connection Cards */}
                    <div className="text-black bg-white rounded-lg shadow-md p-6">
                        <h2 className="text-xl font-semibold mb-4">Connect Services</h2>

                        <div className="space-y-4">
                            <button
                                onClick={() => handleConnectService('spotify')}
                                className="w-full bg-green-600 text-white py-3 px-4 rounded-md hover:bg-green-700 flex items-center justify-center"
                            >
                                <span>Connect Spotify</span>
                            </button>

                            <button
                                onClick={() => handleConnectService('youtube')}
                                className="w-full bg-red-600 text-white py-3 px-4 rounded-md hover:bg-red-700 flex items-center justify-center"
                            >
                                <span>Connect YouTube Music</span>
                            </button>
                        </div>
                    </div>

                    {/* Playlist Transfer */}
                    <div className="bg-white rounded-lg shadow-md p-6">
                        <h2 className="text-xl-gray font-semibold mb-4 text-black">Transfer Playlists</h2>
                        <p className="text-gray-600 mb-4">
                            Connect services to start transferring playlists between platforms.
                        </p>
                        <button
                            disabled
                            className="w-full bg-blue-600 text-white py-3 px-4 rounded-md opacity-50 cursor-not-allowed"
                        >
                            Transfer Playlist (Coming Soon)
                        </button>
                    </div>

                    {/* Connected Services */}
                    <div className="bg-white rounded-lg shadow-md p-6">
                        <h2 className="text-xl font-semibold mb-4 text-black">Connected Services</h2>
                        <div className="space-y-3">
                            <div className="text-black flex items-center justify-between p-3 bg-gray-50 rounded">
                                <span>Spotify</span>
                                <span className="text-red-500 text-sm">Not Connected</span>
                            </div>
                            <div className="text-black flex items-center justify-between p-3 bg-gray-50 rounded">
                                <span>YouTube Music</span>
                                <span className="text-red-500 text-sm">Not Connected</span>
                            </div>
                        </div>
                    </div>
                </div>
            </main>
        </div>
    );
}