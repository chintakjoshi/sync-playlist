import { useEffect } from 'react';
import { useRouter } from 'next/router';

export default function AuthSuccess() {
  const router = useRouter();

  useEffect(() => {
    const { token } = router.query;
    
    if (token && typeof token === 'string') {
      localStorage.setItem('token', token);
      router.push('/dashboard');
    }
  }, [router]);

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-2xl font-bold mb-4">Authentication Successful</h1>
        <p>Redirecting to dashboard...</p>
      </div>
    </div>
  );
}