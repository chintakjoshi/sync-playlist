"use client";

import { useState, useEffect } from 'react';
import axios from 'axios';

interface Playlist {
    id?: number;
    service_id: string;
    name: string;
    description: string;
    track_count: number;
    image_url: string;
    is_public: boolean;
    service_type: string;
}

interface PlaylistsProps {
    service: string;
    isConnected: boolean;
}

export default function Playlists({ service, isConnected }: PlaylistsProps) {
    const [playlists, setPlaylists] = useState<Playlist[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string>('');

    const fetchPlaylists = async () => {
        if (!isConnected) return;

        try {
            setLoading(true);
            setError('');
            const token = localStorage.getItem('token');
            const response = await axios.get(`http://localhost:8080/api/playlists/${service}`, {
                headers: { Authorization: `Bearer ${token}` }
            });
            setPlaylists(response.data.playlists);
        } catch (err: any) {
            console.error(`Failed to fetch ${service} playlists:`, err);
            setError(err.response?.data?.error || 'Failed to fetch playlists');
        } finally {
            setLoading(false);
        }
    };

    const getServiceDisplayName = () => {
        switch (service) {
            case 'spotify': return 'Spotify';
            case 'youtube': return 'YouTube Music';
            default: return service;
        }
    };

    useEffect(() => {
        if (isConnected) {
            fetchPlaylists();
        }
    }, [service, isConnected]);

    if (!isConnected) {
        return (
            <div className="bg-white rounded-lg shadow-md p-6">
                <h3 className="text-lg font-semibold mb-4">{getServiceDisplayName()} Playlists</h3>
                <div className="text-center text-gray-500 py-8">
                    Connect {getServiceDisplayName()} to view playlists
                </div>
            </div>
        );
    }

    return (
        <div className="bg-white rounded-lg shadow-md p-6">
            <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-semibold text-black">{getServiceDisplayName()} Playlists</h3>
                <button
                    onClick={fetchPlaylists}
                    disabled={loading}
                    className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50"
                >
                    {loading ? 'Loading...' : 'Refresh'}
                </button>
            </div>

            {error && (
                <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
                    {error}
                </div>
            )}

            {loading && playlists.length === 0 ? (
                <div className="flex justify-center py-8">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                </div>
            ) : playlists.length === 0 ? (
                <div className="text-center text-gray-500 py-8">
                    No playlists found
                </div>
            ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 max-h-96 overflow-y-auto">
                    {playlists.map((playlist) => (
                        <div
                            key={playlist.service_id}
                            className="border rounded-lg p-4 hover:shadow-md transition-shadow"
                        >
                            <div className="flex items-start space-x-3">
                                {playlist.image_url ? (
                                    <img
                                        src={playlist.image_url}
                                        alt={playlist.name}
                                        className="w-16 h-16 rounded object-cover"
                                    />
                                ) : (
                                    <div className="w-16 h-16 bg-gray-200 rounded flex items-center justify-center">
                                        <span className="text-gray-500 text-xs">No image</span>
                                    </div>
                                )}
                                <div className="flex-1 min-w-0">
                                    <h4 className="font-medium text-gray-900 truncate">
                                        {playlist.name}
                                    </h4>
                                    <p className="text-sm text-gray-500 mt-1">
                                        {playlist.track_count} tracks
                                    </p>
                                    {playlist.description && (
                                        <p className="text-xs text-gray-400 mt-1 truncate">
                                            {playlist.description}
                                        </p>
                                    )}
                                    <div className="flex items-center mt-2">
                                        <span
                                            className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${playlist.is_public
                                                    ? 'bg-green-100 text-green-800'
                                                    : 'bg-gray-100 text-gray-800'
                                                }`}
                                        >
                                            {playlist.is_public ? 'Public' : 'Private'}
                                        </span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}