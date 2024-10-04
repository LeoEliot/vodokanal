<?php

use App\Http\Controllers\API\Auth\LoginController;
use Illuminate\Support\Facades\Route;


Route::get('/',function(){
    return response()->json(['Hello World'=> "Hello World"]);
});

Route::get('/login',LoginController::class);
