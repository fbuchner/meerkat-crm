import { createVuetify } from 'vuetify'
import 'vuetify/styles' // Ensure you are using CSS styles
import { mdi } from 'vuetify/iconsets/mdi' // Use the Material Design Icons (mdi)

const vuetify = createVuetify({
  icons: {
    defaultSet: 'mdi', // Set the default icon pack to mdi
    sets: { mdi },
  },
})

export default vuetify
