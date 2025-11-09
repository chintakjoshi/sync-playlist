"use client";

import { useState, useEffect } from 'react';
import axios from 'axios';

interface Playlist {
    service_id: string;
    name: string;
    description: string;
    track_count: number;
    image_url: string;
    service_type: string;
}

interface TransferModalProps {
    isOpen: boolean;
    onClose: () => void;
    connectedServices: string[];
}

export default function TransferModal({ isOpen, onClose, connectedServices }: TransferModalProps) {
    const [sourceService, setSourceService] = useState('');
    const [targetService, setTargetService] = useState('');
    const [sourcePlaylist, setSourcePlaylist] = useState('');
    const [targetPlaylistName, setTargetPlaylistName] = useState('');
    const [playlists, setPlaylists] = useState<Playlist[]>([]);
    const [loading, setLoading] = useState(false);
    const [playlistsLoading, setPlaylistsLoading] = useState(false);
    const [error, setError] = useState('');

    useEffect(() => {
        const fetchPlaylistsForService = async () => {
            if (!sourceService || !isOpen) {
                setPlaylists([]);
                return;
            }

            try {
                setPlaylistsLoading(true);
                setError('');
                const token = localStorage.getItem('token');
                const response = await axios.get(`http://localhost:8080/api/playlists/${sourceService}`, {
                    headers: { Authorization: `Bearer ${token}` }
                });
                setPlaylists(response.data.playlists || []);
            } catch (err: any) {
                console.error(`Failed to fetch ${sourceService} playlists:`, err);
                setPlaylists([]);
            } finally {
                setPlaylistsLoading(false);
            }
        };

        fetchPlaylistsForService();
    }, [sourceService, isOpen]);

    useEffect(() => {
        if (isOpen) {
            if (connectedServices.length === 2) {
                setSourceService(connectedServices[0]);
                setTargetService(connectedServices[1]);
            }

            setSourcePlaylist('');
            setTargetPlaylistName('');
            setError('');
        } else {
            setSourceService('');
            setTargetService('');
            setSourcePlaylist('');
            setTargetPlaylistName('');
            setPlaylists([]);
            setError('');
        }
    }, [isOpen, connectedServices]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!sourceService || !targetService || !sourcePlaylist) {
            setError('Please fill in all required fields');
            return;
        }

        try {
            setLoading(true);
            setError('');
            const token = localStorage.getItem('token');

            const response = await axios.post('http://localhost:8080/api/transfers', {
                source_service: sourceService,
                source_playlist_id: sourcePlaylist,
                target_service: targetService,
                target_playlist_name: targetPlaylistName,
            }, {
                headers: { Authorization: `Bearer ${token}` }
            });

            alert(`Transfer started! Transfer ID: ${response.data.transfer_id}`);
            onClose();
        } catch (err: any) {
            console.error('Transfer failed:', err);
            setError(err.response?.data?.error || 'Transfer failed');
        } finally {
            setLoading(false);
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 w-full max-w-md">
                <h2 className="text-xl font-bold mb-4 text-black">Transfer Playlist</h2>

                <form onSubmit={handleSubmit} className="space-y-4">
                    {/* Source Service */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">
                            From Service *
                        </label>
                        <select
                            value={sourceService}
                            onChange={(e) => {
                                setSourceService(e.target.value);
                                setSourcePlaylist('');
                            }}
                            className="text-black w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                            required
                        >
                            <option value="">Select source service</option>
                            {connectedServices.map(service => (
                                <option key={service} value={service}>
                                    {service === 'spotify' ? 'Spotify' : 'YouTube Music'}
                                </option>
                            ))}
                        </select>
                    </div>

                    {/* Source Playlist */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">
                            Playlist to Transfer *
                        </label>
                        <select
                            value={sourcePlaylist}
                            onChange={(e) => setSourcePlaylist(e.target.value)}
                            className="text-black w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                            required
                            disabled={!sourceService || playlistsLoading}
                        >
                            <option value="">Select playlist</option>
                            {playlistsLoading ? (
                                <option value="" disabled>Loading playlists...</option>
                            ) : playlists.length === 0 ? (
                                <option value="" disabled>No playlists found</option>
                            ) : (
                                playlists.map(playlist => (
                                    <option key={playlist.service_id} value={playlist.service_id}>
                                        {playlist.name} ({playlist.track_count} tracks)
                                    </option>
                                ))
                            )}
                        </select>
                        {playlistsLoading && (
                            <p className="text-sm text-gray-500 mt-1">Loading playlists...</p>
                        )}
                    </div>

                    {/* Target Service */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">
                            To Service *
                        </label>
                        <select
                            value={targetService}
                            onChange={(e) => setTargetService(e.target.value)}
                            className="text-black w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                            required
                        >
                            <option value="">Select target service</option>
                            {connectedServices
                                .filter(service => service !== sourceService)
                                .map(service => (
                                    <option key={service} value={service}>
                                        {service === 'spotify' ? 'Spotify' : 'YouTube Music'}
                                    </option>
                                ))
                            }
                        </select>
                    </div>

                    {/* Target Playlist Name */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">
                            New Playlist Name
                        </label>
                        <input
                            type="text"
                            value={targetPlaylistName}
                            onChange={(e) => setTargetPlaylistName(e.target.value)}
                            placeholder="Leave empty to use original name"
                            className="text-black w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                    </div>

                    {error && (
                        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                            {error}
                        </div>
                    )}

                    <div className="flex justify-end space-x-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 text-gray-600 hover:text-gray-800"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={loading}
                            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50"
                        >
                            {loading ? 'Starting Transfer...' : 'Start Transfer'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}