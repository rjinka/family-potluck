import React from 'react';
import { GoogleLogin } from '@react-oauth/google';
import { useAuth } from '../context/AuthContext';
import { useNavigate } from 'react-router-dom';
import { UtensilsCrossed } from 'lucide-react';

const Login = () => {
    const { user, login } = useAuth();
    const navigate = useNavigate();

    React.useEffect(() => {
        if (user) {
            navigate('/dashboard');
        }
    }, [user, navigate]);

    const handleSuccess = async (credentialResponse) => {
        try {
            await login(credentialResponse.credential);
            navigate('/groups');
        } catch (error) {
            console.error("Login Failed", error);
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
            </div>
        </div>
    );
};

export default Login;
