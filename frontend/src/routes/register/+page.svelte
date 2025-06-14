<script lang="ts">
  import { goto } from '$app/navigation';
  import userService from '$lib/services/userService';

  let username = '';
  let email = '';
  let password = '';
  let errorMessage = '';
  let loading = false;

  async function handleRegister() {
    loading = true;
    errorMessage = '';
    
    try {
      await userService.register({ username, email, password });
      goto('/login'); // Redirect to login after successful registration
    } catch (error) {
      errorMessage = error instanceof Error 
        ? error.message 
        : 'Registration failed. Please try again.';
    } finally {
      loading = false;
    }
  }
</script>

<div class="max-w-md mx-auto mt-10 p-6 bg-white rounded-lg shadow-md">
  <h1 class="text-2xl font-bold mb-6 text-center">Register</h1>
  
  <form on:submit|preventDefault={handleRegister} class="space-y-4">
    <div>
      <label for="username" class="block text-sm font-medium text-gray-700">Username</label>
      <input
        id="username"
        type="text"
        bind:value={username}
        required
        class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
      />
    </div>
    
    <div>
      <label for="email" class="block text-sm font-medium text-gray-700">Email</label>
      <input
        id="email"
        type="email"
        bind:value={email}
        required
        class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
      />
    </div>
    
    <div>
      <label for="password" class="block text-sm font-medium text-gray-700">Password</label>
      <input
        id="password"
        type="password"
        bind:value={password}
        required
        class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
      />
    </div>
    
    <div>
      <button
        type="submit"
        disabled={loading}
        class="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
      >
        {loading ? 'Registering...' : 'Register'}
      </button>
    </div>
    
    {#if errorMessage}
      <div class="p-3 bg-red-100 text-red-700 rounded-md text-sm">
        {errorMessage}
      </div>
    {/if}
    
    <div class="text-center mt-4">
      <p class="text-sm text-gray-600">
        Already have an account? 
        <a href="/login" class="font-medium text-indigo-600 hover:text-indigo-500">
          Login
        </a>
      </p>
    </div>
  </form>
</div>
