"use client";

import { useState, useEffect } from 'react';
import axios from 'axios';

interface Transfer {
    id: number;
    source_service: string;
    source_playlist_name: string;
    target_service: string;
    target_playlist_name: string;
    status: string;
    tracks_total: number;
    tracks_matched: number;
    tracks_failed: number;
    created_at: string;
    updated_at: string;
}

interface TransferTrack {
    id: number;
    source_track_id: string;
    source_track_name: string;
    source_artist: string;
    target_track_id: string;
    target_track_name: string;
    target_artist: string;
    status: string;
    match_confidence: number;
}

export default function TransferHistory() {
    const [transfers, setTransfers] = useState<Transfer[]>([]);
    const [selectedTransfer, setSelectedTransfer] = useState<Transfer | null>(null);
    const [transferTracks, setTransferTracks] = useState<TransferTrack[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string>('');

    const fetchTransfers = async () => {
        try {
            setLoading(true);
            setError('');
            const token = localStorage.getItem('token');
            const response = await axios.get('http://localhost:8080/api/transfers', {
                headers: { Authorization: `Bearer ${token}` }
            });
            setTransfers(response.data.transfers);
        } catch (error: any) {
            console.error('Failed to fetch transfers:', error);
            setError(error.response?.data?.error || 'Failed to fetch transfers');
        } finally {
            setLoading(false);
        }
    };

    const fetchTransferDetails = async (transferId: number) => {

        try {
            setError('');
            const token = localStorage.getItem('token');
            const response = await axios.get(`http://localhost:8080/api/transfers/${transferId}`, {
                headers: { Authorization: `Bearer ${token}` }
            });
            setTransferTracks(response.data.tracks || []);
            setSelectedTransfer(response.data.transfer);
        } catch (error: any) {
            console.error('Failed to fetch transfer details:', error);
            if (error.response?.status === 404) {
                setError('Transfer not found. It might have been deleted.');
            } else {
                setError(error.response?.data?.error || 'Failed to fetch transfer details');
            }
        }
    };

    useEffect(() => {
        fetchTransfers();
    }, []);

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'completed':
                return 'bg-green-100 text-green-800';
            case 'completed_with_errors':
                return 'bg-yellow-100 text-yellow-800';
            case 'failed':
                return 'bg-red-100 text-red-800';
            case 'processing':
                return 'bg-blue-100 text-blue-800';
            case 'pending':
                return 'bg-gray-100 text-gray-800';
            default:
                return 'bg-gray-100 text-gray-800';
        }
    };

    const formatDate = (dateString: string) => {
        if (!dateString) return 'Unknown date';

        try {
            // Handle both ISO string and other formats
            const date = new Date(dateString);
            if (isNaN(date.getTime())) {
                // Try parsing as Unix timestamp if it's a number
                const timestamp = parseInt(dateString);
                if (!isNaN(timestamp)) {
                    return new Date(timestamp * 1000).toLocaleString();
                }
                return 'Invalid date';
            }
            return date.toLocaleString();
        } catch (error) {
            console.error('Date parsing error:', error, dateString);
            return 'Date error';
        }
    };

    const getStatusDisplay = (status: string) => {
        const statusMap: { [key: string]: string } = {
            'completed': 'Completed',
            'completed_with_errors': 'Completed with errors',
            'failed': 'Failed',
            'processing': 'Processing',
            'pending': 'Pending'
        };
        return statusMap[status] || status;
    };

    return (
        <div className="bg-white rounded-lg shadow-md p-6">
            <div className="flex justify-between items-center mb-4">
                <h2 className="text-xl font-semibold text-black">Transfer History</h2>
                <button
                    onClick={fetchTransfers}
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

            {transfers.length === 0 ? (
                <div className="text-center text-gray-500 py-8">
                    No transfers yet
                </div>
            ) : (
                <div className="space-y-4">
                    {transfers.map(transfer => (
                        <div
                            key={transfer.id}
                            className="border rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer"
                            onClick={() => {
                                fetchTransferDetails(transfer.id);
                            }}
                        >
                            <div className="flex justify-between items-start">
                                <div className="flex-1">
                                    <h3 className="font-medium text-lg text-black">
                                        {transfer.source_playlist_name || 'Unknown playlist'} → {transfer.target_playlist_name || 'Unknown playlist'}
                                    </h3>
                                    <p className="text-sm text-gray-500 mt-1">
                                        {transfer.source_service} → {transfer.target_service}
                                    </p>
                                    <p className="text-sm text-gray-600 mt-2">
                                        {transfer.tracks_matched}/{transfer.tracks_total} tracks transferred
                                        {transfer.tracks_failed > 0 && (
                                            <span className="text-red-600"> ({transfer.tracks_failed} failed)</span>
                                        )}
                                    </p>
                                </div>
                                <div className="flex flex-col items-end space-y-2 ml-4">
                                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(transfer.status)}`}>
                                        {getStatusDisplay(transfer.status)}
                                    </span>
                                    <span className="text-xs text-gray-500 text-right">
                                        {formatDate(transfer.created_at)}
                                    </span>
                                </div>
                            </div>

                            {/* Transfer details */}
                            {selectedTransfer?.id === transfer.id && (
                                <div className="mt-4 border-t pt-4">
                                    <h4 className="font-medium mb-3 text-gray-700">Track Details:</h4>
                                    {transferTracks.length === 0 ? (
                                        <div className="text-center text-gray-500 py-4">
                                            No track details available
                                        </div>
                                    ) : (
                                        <>
                                            <div className="max-h-60 overflow-y-auto space-y-2">
                                                {transferTracks.map((track, index) => (
                                                    <div
                                                        key={track.id || `track-${index}`} // Fallback to index if id is missing
                                                        className={`flex justify-between items-center py-2 px-3 rounded ${track.status === 'matched' ? 'bg-green-50 border border-green-200' :
                                                            track.status === 'not_found' ? 'bg-red-50 border border-red-200' :
                                                                'bg-yellow-50 border border-yellow-200'
                                                            }`}
                                                    >
                                                        <div className="flex-1 min-w-0">
                                                            <div className="font-medium text-sm truncate">
                                                                {track.source_track_name || 'Unknown track'}
                                                            </div>
                                                            <div className="text-xs text-gray-500 truncate">
                                                                {track.source_artist || 'Unknown artist'}
                                                            </div>
                                                        </div>

                                                        <div className="mx-3 text-gray-400 flex-shrink-0">→</div>

                                                        <div className="flex-1 min-w-0">
                                                            {track.status === 'matched' ? (
                                                                <>
                                                                    <div className="font-medium text-sm truncate">
                                                                        {track.target_track_name || 'Unknown track'}
                                                                    </div>
                                                                    <div className="text-xs text-gray-500 truncate">
                                                                        {track.target_artist || 'Unknown artist'}
                                                                    </div>
                                                                </>
                                                            ) : (
                                                                <div className="text-red-600 text-sm">Not found</div>
                                                            )}
                                                        </div>

                                                        <div className="text-xs text-gray-500 ml-3 flex-shrink-0">
                                                            {track.status === 'matched' && `Match: ${Math.round(track.match_confidence * 100)}%`}
                                                            {track.status === 'not_found' && 'No match'}
                                                            {track.status === 'error' && 'Error'}
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>

                                            {/* Summary */}
                                            <div className="mt-3 text-sm text-gray-600">
                                                <p>
                                                    <span className="font-medium">Summary:</span> {transferTracks.filter(t => t.status === 'matched').length} matched,
                                                    {' '}{transferTracks.filter(t => t.status === 'not_found').length} not found,
                                                    {' '}{transferTracks.filter(t => t.status === 'error').length} errors
                                                </p>
                                            </div>
                                        </>
                                    )}
                                </div>
                            )}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}