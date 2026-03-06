"use client";

import { Suspense, useEffect, useState } from "react";
import Image from "next/image";
import { useRouter, useSearchParams } from "next/navigation";

import Playlists from "../components/Playlists";
import TransferModal from "../components/TransferModal";
import TransferHistory from "../components/TransferHistory";
import { api, getApiErrorMessage } from "../../lib/api";

interface User {
  id: number;
  email: string;
  name: string;
  avatarURL?: string;
}

interface ConnectedService {
  id: number;
  user_id: number;
  service_type: string;
  service_user_name: string;
  created_at: string;
}

function DashboardContent() {
  const [user, setUser] = useState<User | null>(null);
  const [unlinkingService, setUnlinkingService] = useState<string | null>(null);
  const [connectedServices, setConnectedServices] = useState<ConnectedService[]>([]);
  const [loading, setLoading] = useState(true);
  const [servicesLoading, setServicesLoading] = useState(false);
  const [message, setMessage] = useState<string>("");
  const [showTransferModal, setShowTransferModal] = useState(false);
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    checkAuth();

    // Check for success messages from OAuth redirects
    const messageParam = searchParams.get('message');
    if (messageParam) {
      if (messageParam === 'spotify_connected') {
        setMessage("Successfully connected to Spotify!");
        fetchConnectedServices();
      } else if (messageParam === 'youtube_connected') {
        setMessage("Successfully connected to YouTube Music!");
        fetchConnectedServices();
      } else if (messageParam === "login_success") {
        setMessage("Successfully logged in.");
        fetchConnectedServices();
      }

      setTimeout(() => setMessage(""), 5000);
    }
    // Dashboard auth/message bootstrapping is intentionally driven by search params.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const checkAuth = async () => {
    try {
      const response = await api.get("/api/auth/me");
      setUser(response.data.user);

      fetchConnectedServices();
    } catch {
      router.push("/");
    } finally {
      setLoading(false);
    }
  };

  const fetchConnectedServices = async () => {
    try {
      setServicesLoading(true);
      const response = await api.get("/api/services");
      setConnectedServices(response.data.services);
    } catch (error: unknown) {
      console.error("Failed to fetch connected services:", error);
      if (
        typeof error === "object" &&
        error !== null &&
        "response" in error &&
        (error as { response?: { status?: number } }).response?.status === 401
      ) {
        router.push("/");
      }
    } finally {
      setServicesLoading(false);
    }
  };

  const handleConnectService = async (provider: string) => {
    try {
      const response = await api.post(`/api/services/connect/${provider}`);
      window.location.href = response.data.auth_url;
    } catch (error) {
      console.error("Failed to connect service:", error);
    }
  };

  const handleUnlinkService = async (serviceType: string) => {
    if (!confirm(`Are you sure you want to disconnect ${getServiceDisplayName(serviceType)}? This will remove all associated playlists from our system.`)) {
      return;
    }

    try {
      setUnlinkingService(serviceType);
      await api.delete(`/api/services/${serviceType}`);

      fetchConnectedServices();

      setMessage(`Successfully disconnected ${getServiceDisplayName(serviceType)}`);
      setTimeout(() => setMessage(""), 5000);
    } catch (error: unknown) {
      console.error('Failed to disconnect service:', error);
      alert(getApiErrorMessage(error, 'Failed to disconnect service'));
    } finally {
      setUnlinkingService(null);
    }
  };

  const handleLogout = async () => {
    try {
      await api.post("/api/auth/logout");
    } catch (error) {
      console.error("Logout error:", error);
    } finally {
      router.push("/");
    }
  };

  const isServiceConnected = (serviceType: string) => {
    return connectedServices.some(service => service.service_type === serviceType);
  };

  const getServiceDisplayName = (serviceType: string) => {
    switch (serviceType) {
      case 'spotify': return 'Spotify';
      case 'youtube': return 'YouTube Music';
      default: return serviceType;
    }
  };

  const getServiceUserName = (serviceType: string) => {
    const service = connectedServices.find(s => s.service_type === serviceType);

    if (!service) return 'Not Connected';

    // Return the actual username from the service, or a default
    return service.service_user_name || 'Connected';
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
                  <Image
                    src={user.avatarURL}
                    alt={user.name}
                    width={32}
                    height={32}
                    className="w-8 h-8 rounded-full"
                    unoptimized
                  />
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
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Left Column - Services & Transfer */}
          <div className="space-y-6">
            {/* Service Connection Cards */}
            <div className="bg-white rounded-lg shadow-md p-6">
              <h2 className="text-xl font-semibold mb-4 text-black">Connect Services</h2>

              <div className="space-y-4">
                <button
                  onClick={() => handleConnectService('spotify')}
                  disabled={isServiceConnected('spotify')}
                  className={`w-full py-3 px-4 rounded-md flex items-center justify-center ${isServiceConnected('spotify')
                    ? 'bg-green-500 text-white opacity-50 cursor-not-allowed'
                    : 'bg-green-600 text-white hover:bg-green-700'
                    }`}
                >
                  <span>
                    {isServiceConnected('spotify') ? 'Spotify Connected' : 'Connect Spotify'}
                  </span>
                </button>

                <button
                  onClick={() => handleConnectService('youtube')}
                  disabled={isServiceConnected('youtube')}
                  className={`w-full py-3 px-4 rounded-md flex items-center justify-center ${isServiceConnected('youtube')
                    ? 'bg-red-500 text-white opacity-50 cursor-not-allowed'
                    : 'bg-red-600 text-white hover:bg-red-700'
                    }`}
                >
                  <span>
                    {isServiceConnected('youtube') ? 'YouTube Music Connected' : 'Connect YouTube Music'}
                  </span>
                </button>
              </div>
            </div>

            {/* Playlist Transfer */}
            <div className="bg-white rounded-lg shadow-md p-6">
              <h2 className="text-xl font-semibold mb-4 text-black">Transfer Playlists</h2>
              <p className="text-gray-600 mb-4">
                Select playlists to transfer between services.
              </p>
              <button
                onClick={() => setShowTransferModal(true)}
                disabled={connectedServices.length < 2}
                className={`w-full py-3 px-4 rounded-md ${connectedServices.length < 2
                  ? 'bg-blue-400 text-white opacity-50 cursor-not-allowed'
                  : 'bg-blue-600 text-white hover:bg-blue-700'
                  }`}
              >
                {connectedServices.length < 2
                  ? 'Connect at least 2 services'
                  : 'Transfer Playlist'
                }
              </button>
            </div>

            {/* Connected Services */}
            <div className="bg-white rounded-lg shadow-md p-6">
              <h2 className="text-xl font-semibold mb-4 text-black">Connected Services</h2>
              {servicesLoading ? (
                <div className="flex justify-center">
                  <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
                </div>
              ) : (
                <div className="space-y-3 text-black">
                  {['spotify', 'youtube'].map((service) => (
                    <div key={service} className="flex items-center justify-between p-3 bg-gray-50 rounded">
                      <div className="flex-1">
                        <div className="flex items-center justify-between">
                          <div>
                            <span className="font-medium">{getServiceDisplayName(service)}</span>
                            {isServiceConnected(service) && (
                              <p className="text-xs text-gray-500">{getServiceUserName(service)}</p>
                            )}
                          </div>
                          <div className="flex items-center space-x-2">
                            {isServiceConnected(service) ? (
                              <>
                                <span className="text-green-500 text-sm">Connected</span>
                                <button
                                  onClick={() => handleUnlinkService(service)}
                                  disabled={unlinkingService === service}
                                  className="text-red-600 hover:text-red-800 text-sm font-medium px-3 py-1 border border-red-300 rounded hover:bg-red-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                >
                                  {unlinkingService === service ? 'Unlinking...' : 'Unlink'}
                                </button>
                              </>
                            ) : (
                              <span className="text-red-500 text-sm">Not Connected</span>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}

                  {connectedServices.length === 0 && (
                    <p className="text-gray-500 text-center py-4">No services connected yet</p>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Right Column - Playlists */}
          <div className="space-y-6">
            <Playlists service="spotify" isConnected={isServiceConnected('spotify')} />
            <Playlists service="youtube" isConnected={isServiceConnected('youtube')} />
          </div>
        </div>
      </main>
      <TransferModal
        isOpen={showTransferModal}
        onClose={() => setShowTransferModal(false)}
        connectedServices={connectedServices.map(s => s.service_type)}
      />
      <div className="mt-8">
        <TransferHistory />
      </div>
    </div>
  );
}

export default function Dashboard() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen flex items-center justify-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      }
    >
      <DashboardContent />
    </Suspense>
  );
}
