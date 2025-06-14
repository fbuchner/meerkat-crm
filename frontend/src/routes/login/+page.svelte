<script lang="ts">
  import { goto } from '$app/navigation';
  import userService from '$lib/services/userService';
  import { userStore } from '$lib/stores/userStore';

  let email = '';
  let password = '';
  let errorMessage = '';
  let loading = false;

  async function handleLogin() {
    loading = true;
    errorMessage = '';
    
    try {
      await userService.login({ email, password });
      goto('/dashboard'); // Redirect to dashboard after successful login
    } catch (error) {
      errorMessage = error instanceof Error 
        ? error.message 
        : 'Login failed. Please try again.';
    } finally {
      loading = false;
    }
  }
</script>

<div class="max-w-md mx-auto mt-10 p-6 bg-white rounded-lg shadow-md">
  <h1 class="text-2xl font-bold mb-6 text-center">Login</h1>
  
  <form on:submit|preventDefault={handleLogin} class="space-y-4">
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
        {loading ? 'Logging in...' : 'Login'}
      </button>
    </div>
    
    {#if errorMessage}
      <div class="p-3 bg-red-100 text-red-700 rounded-md text-sm">
        {errorMessage}
      </div>
    {/if}
    
    <div class="text-center mt-4">
      <p class="text-sm text-gray-600">
        Don't have an account? 
        <a href="/register" class="font-medium text-indigo-600 hover:text-indigo-500">
          Register
        </a>
      </p>
    </div>
  </form>
</div>
