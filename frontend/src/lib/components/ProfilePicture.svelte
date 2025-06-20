<script lang="ts">
  import { onMount } from 'svelte';
  import { Avatar } from 'flowbite-svelte';
  import { PUBLIC_API_URL } from '$env/static/public';
  
  export let contactId: number | string | undefined = undefined;
  export let photo: string | undefined = undefined; 
  export let initials: string = '';
  export let size: 'xs' | 'sm' | 'md' | 'lg' | 'xl' = 'lg';
  export let cornerStyle: 'rounded' | 'circular' = 'rounded';
  export let styleclass: string = '';
  
  let imageSrc: string | undefined = undefined;
  let loading = false;
  let error = false;
  
  // Default placeholder image
  const placeholderImage = '/assets/placeholder-avatar.png';
  
  // Fetch the profile picture if contactId is provided and no photo URL is passed
  onMount(async () => {
    if (contactId) {
      await fetchProfilePicture();
    }
  });
  
  async function fetchProfilePicture() {
    if (!contactId) return;
    
    loading = true;
    error = false;
    
    try {
      // Get the token from localStorage
      const token = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : null;
      
      // Format the URL properly with the API base URL
      let url = `${PUBLIC_API_URL}/contacts/${contactId}/profile_picture`;
      // Remove any potential double slashes except in http(s)://
      url = url.replace(/([^:]\/)\/+/g, "$1");
      
      // Fetch the image using fetch API with Authorization header
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          // Add the header if token exists
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch profile picture');
      }
      
      const blob = await response.blob();
      imageSrc = URL.createObjectURL(blob);
    } catch (err) {
      console.error('Error fetching profile picture:', err);
      error = true;
      // Use the provided photo URL as fallback, or the placeholder
      imageSrc = photo || placeholderImage;
    } finally {
      loading = false;
    }
  }
</script>

{#if loading}
  <Avatar {size} {cornerStyle} class={`${styleclass}`}>
    <svg aria-hidden="true" class="animate-spin" viewBox="0 0 100 101" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M100 50.5908C100 78.2051 77.6142 100.591 50 100.591C22.3858 100.591 0 78.2051 0 50.5908C0 22.9766 22.3858 0.59082 50 0.59082C77.6142 0.59082 100 22.9766 100 50.5908ZM9.08144 50.5908C9.08144 73.1895 27.4013 91.5094 50 91.5094C72.5987 91.5094 90.9186 73.1895 90.9186 50.5908C90.9186 27.9921 72.5987 9.67226 50 9.67226C27.4013 9.67226 9.08144 27.9921 9.08144 50.5908Z" fill="#E5E7EB"/>
      <path d="M93.9676 39.0409C96.393 38.4038 97.8624 35.9116 97.0079 33.5539C95.2932 28.8227 92.871 24.3692 89.8167 20.348C85.8452 15.1192 80.8826 10.7238 75.2124 7.41289C69.5422 4.10194 63.2754 1.94025 56.7698 1.05124C51.7666 0.367541 46.6976 0.446843 41.7345 1.27873C39.2613 1.69328 37.813 4.19778 38.4501 6.62326C39.0873 9.04874 41.5694 10.4717 44.0505 10.1071C47.8511 9.54855 51.7191 9.52689 55.5402 10.0491C60.8642 10.7766 65.9928 12.5457 70.6331 15.2552C75.2735 17.9648 79.3347 21.5619 82.5849 25.841C84.9175 28.9121 86.7997 32.2913 88.1811 35.8758C89.083 38.2158 91.5421 39.6781 93.9676 39.0409Z" fill="currentColor"/>
    </svg>
  </Avatar>
{:else if imageSrc}
  <Avatar src={imageSrc} {size} {cornerStyle} class={`${styleclass}`} />
{:else if photo}
  <Avatar src={photo} {size} {cornerStyle} class={`${styleclass}`} />
{:else}
  <Avatar {size} {cornerStyle} class={`${styleclass}`}>{initials}</Avatar>
{/if}
