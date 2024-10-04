<?php

namespace App\Http\Controllers\API\Auth;

use App\Http\Controllers\Controller;
use Illuminate\Http\Request;
use Firebase\JWT\JWT;
use Firebase\JWT\Key;


class LoginController extends Controller
{
    public function __invoke()
    {
        $jwt_key = env('JWT_SECRET_KEY','qwerty');
        $payload = [
            'username' => 'Viktor',
            'password' => '123456'
        ];

        $encode = JWT::encode($payload, $jwt_key, 'HS256');
        $decode = JWT::decode($encode,new Key($jwt_key, 'HS256'));
        return response()->json([
            'access_token' => $encode,
            'decoded' => $decode
        ]);
    }
}
