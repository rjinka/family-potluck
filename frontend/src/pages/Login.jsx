import React from 'react';
import { GoogleLogin } from '@react-oauth/google';
import { useAuth } from '../context/AuthContext';
import { useNavigate } from 'react-router-dom';
import { UtensilsCrossed } from 'lucide-react';

const Login = () => {
    const { login, devLogin } = useAuth();
    const navigate = useNavigate();

    const handleSuccess = async (credentialResponse) => {
        try {
            await login(credentialResponse.credential);
            navigate('/groups');
        } catch (error) {
            console.error("Login Failed", error);
        }
    };

    const handleDevLogin = async (email, name) => {
        try {
            await devLogin(email, name);
            navigate('/groups');
        } catch (error) {
            console.error("Dev Login Failed", error);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-orange-100 to-amber-50">
            <div className="bg-white p-8 rounded-2xl shadow-xl w-full max-w-md text-center">
                <div className="flex justify-center mb-6">
                    <div className="bg-orange-500 p-3 rounded-full">
                        <UtensilsCrossed className="w-10 h-10 text-white" />
                    </div>
                </div>
                <h1 className="text-3xl font-bold text-gray-800 mb-2">Family Gatherings</h1>
                <p className="text-gray-500 mb-8">Gather, Share, and Enjoy!</p>

                <div className="flex justify-center">
                    <GoogleLogin
                        onSuccess={handleSuccess}
                        onError={() => {
                            console.log('Login Failed');
                        }}
                        useOneTap
                        shape="pill"
                        size="large"
                    />
                </div>
                <p className="mt-6 text-sm text-gray-400">
                    Sign in with Google to join your family group.
                </p>

                {/* Dev Login */}
                <div className="mt-8 border-t pt-6">
                    <h3 className="text-sm font-semibold text-gray-500 mb-4">Dev Login</h3>
                    <form onSubmit={(e) => {
                        e.preventDefault();
                        const email = e.target.email.value;
                        const name = e.target.name.value;
                        handleDevLogin(email, name);
                    }} className="space-y-3">
                        <input name="email" placeholder="Email" className="w-full p-2 border rounded" required />
                        <input name="name" placeholder="Name" className="w-full p-2 border rounded" required />
                        <button type="submit" className="w-full bg-gray-800 text-white py-2 rounded hover:bg-gray-700">
                            Dev Login
                        </button>
                    </form>
                </div>
            </div>
        </div>
    );
};

export default Login;
