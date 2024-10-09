<template>
  <div id="app">
    <nav>
      <ul>
        <li><router-link to="/contacts">Contact List</router-link></li>
        <li><router-link to="/add-contact">Add Contact</router-link></li>
      </ul>
    </nav>
    <router-view></router-view> <!-- Router will inject the matched component here -->
  </div>
</template>

<script>
export default {
  name: 'App',
};
</script>

<style scoped>
/* Add some basic styles for the navigation */
nav ul {
  list-style-type: none;
  padding: 0;
}

nav ul li {
  display: inline;
  margin-right: 15px;
}

nav ul li a {
  text-decoration: none;
  color: #007bff;
}

nav ul li a:hover {
  color: #0056b3;
}
</style>
